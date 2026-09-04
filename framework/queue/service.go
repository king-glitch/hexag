package queue

import (
	"context"
	"math"
	"time"

	"github.com/king-glitch/hexag/framework/ports"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	defaultDequeueLimit     = 100
	defaultMaxAttempts      = 5
	defaultRetryBaseBackoff = 2 * time.Second
	defaultMaxRetryBackoff  = 10 * time.Minute
)

type Option func(*Service)

func WithDequeueLimit(limit int) Option {
	return func(s *Service) { s.dequeueLimit = limit }
}

func WithMaxAttempts(attempts int) Option {
	return func(s *Service) { s.maxAttempts = attempts }
}

func WithRetryBackoff(base time.Duration, max time.Duration) Option {
	return func(s *Service) {
		s.retryBaseBackoff = base
		s.maxRetryBackoff = max
	}
}

type Service struct {
	ports.ServiceBase

	tag         string
	repository  ports.QueueRepository
	transformer ports.Transformer

	dequeueLimit     int
	maxAttempts      int
	retryBaseBackoff time.Duration
	maxRetryBackoff  time.Duration
}

func NewService(
	ctx ports.ServiceContext,
	tag string,
	repository ports.QueueRepository,
	transformer ports.Transformer,
	options ...Option,
) ports.QueueService {
	s := Service{
		ServiceBase:      ports.NewBaseService(ctx),
		tag:              tag,
		repository:       repository,
		transformer:      transformer,
		dequeueLimit:     defaultDequeueLimit,
		maxAttempts:      defaultMaxAttempts,
		retryBaseBackoff: defaultRetryBaseBackoff,
		maxRetryBackoff:  defaultMaxRetryBackoff,
	}

	for _, apply := range options {
		apply(&s)
	}

	return s
}

func (q Service) GetTag() string {
	return q.tag
}

func (q Service) retryBackoff(attempts int) time.Duration {
	if attempts <= 0 {
		return q.retryBaseBackoff
	}

	multiplier := math.Pow(2, float64(attempts-1))
	backoff := time.Duration(float64(q.retryBaseBackoff) * multiplier)
	if backoff > q.maxRetryBackoff {
		return q.maxRetryBackoff
	}

	return backoff
}

func (q Service) Enqueue(ctx context.Context, item ports.QueueItem) (bson.ObjectID, error) {
	now := time.Now()

	id, err := q.repository.InsertItem(
		ctx,
		ports.QueueModel{
			Id:          bson.NewObjectID(),
			Data:        item.GetData(),
			ActionType:  item.GetActionType(),
			Tag:         q.tag,
			Status:      ports.ItemStatusPending,
			Priority:    item.GetPriority(),
			ScheduledAt: item.GetScheduledAt(),
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	)
	if err != nil {
		return bson.ObjectID{}, errors.Wrap(err, "failed to enqueue item")
	}

	return id, nil
}

func (q Service) logger() *zerolog.Logger {
	if q.GetContext() != nil && q.GetContext().Logger() != nil {
		return q.GetContext().Logger()
	}

	nop := zerolog.Nop()
	return &nop
}

func (q Service) dequeue(
	ctx context.Context,
	dequeueOptions []ports.DequeueOption,
	process func(item ports.QueueModel, transformed ports.QueueItem) (ports.QueueItemStatus, error),
) error {
	at := time.Now()
	limit := q.dequeueLimit

	var option ports.DequeueOptionData
	for _, apply := range dequeueOptions {
		apply(&option)
	}

	if option.At != nil {
		at = *option.At
	}

	if option.Limit != nil {
		limit = *option.Limit
	}

	items, err := q.repository.ClaimReadyItems(ctx, q.tag, at, limit)
	if err != nil {
		return errors.Wrap(err, "failed to claim ready queue items")
	}

	for _, item := range items {
		transformed, err := q.transformer.Transform(item)
		if err != nil {
			q.logger().Error().Err(err).Str("id", item.Id.Hex()).Msg("failed to transform queue item")
			_ = q.repository.UpdateStatus(ctx, item.Id, ports.ItemStatusFailed, err.Error())

			continue
		}

		status, err := process(item, transformed)
		if err != nil {
			q.logger().Error().Err(err).Str("id", item.Id.Hex()).Msg("failed to process queue item")
			_ = q.repository.UpdateStatus(ctx, item.Id, ports.ItemStatusFailed, err.Error())

			continue
		}

		if status == ports.ItemStatusRetry {
			if item.Attempts >= q.maxAttempts {
				q.logger().Warn().
					Str("id", item.Id.Hex()).
					Int("attempts", item.Attempts).
					Msg("max retry attempts reached for queue item")

				_ = q.repository.UpdateStatus(ctx, item.Id, ports.ItemStatusFailed, "max retry attempts reached")
			} else {
				nextSchedule := at.Add(q.retryBackoff(item.Attempts))
				_ = q.repository.RescheduleItem(ctx, item.Id, nextSchedule, "")
			}

			continue
		}

		if err := q.repository.UpdateStatus(ctx, item.Id, status, ""); err != nil {
			q.logger().Error().Err(err).Str("id", item.Id.Hex()).Msg("failed to update queue item status")
		}
	}

	return nil
}

func (q Service) Dequeue(
	ctx context.Context,
	f func(ports.QueueItem) error,
	dequeueOptions ...ports.DequeueOption,
) error {
	if err := q.dequeue(
		ctx,
		dequeueOptions,
		func(item ports.QueueModel, transformed ports.QueueItem) (ports.QueueItemStatus, error) {
			if err := f(transformed); err != nil {
				return "", errors.Wrap(err, "user queue callback failed")
			}

			return ports.ItemStatusCompleted, nil
		},
	); err != nil {
		return errors.Wrap(err, "failed to dequeue items")
	}

	return nil
}

func (q Service) DequeueWithManualAck(
	ctx context.Context,
	f func(ports.QueueItem) (ports.QueueItemStatus, error),
	dequeueOptions ...ports.DequeueOption,
) error {
	if err := q.dequeue(
		ctx,
		dequeueOptions,
		func(item ports.QueueModel, transformed ports.QueueItem) (ports.QueueItemStatus, error) {
			q.logger().Info().
				Str("item_id", item.Id.Hex()).
				Str("action_type", string(item.ActionType)).
				Msg("processing queue item")

			status, err := f(transformed)
			if err != nil {
				q.logger().Error().
					Err(err).
					Str("item_id", item.Id.Hex()).
					Msg("failed to process queue item")

				return "", errors.Wrap(err, "user queue callback failed")
			}

			return status, nil
		},
	); err != nil {
		return errors.Wrap(err, "failed to dequeue items with manual ack")
	}

	return nil
}

func (q Service) Cancel(ctx context.Context, id bson.ObjectID, reason ...string) error {
	message := ""
	if len(reason) > 0 {
		message = reason[0]
	}

	if err := q.repository.UpdateStatus(ctx, id, ports.ItemStatusCanceled, message); err != nil {
		return errors.Wrap(err, "failed to cancel queue item")
	}

	return nil
}

func (q Service) BulkCancel(ctx context.Context, ids []bson.ObjectID, reason ...string) error {
	message := ""
	if len(reason) > 0 {
		message = reason[0]
	}

	if err := q.repository.UpdateManyStatus(ctx, ids, ports.ItemStatusCanceled, message); err != nil {
		return errors.Wrap(err, "failed to bulk cancel queue items")
	}

	return nil
}

func (q Service) UpdateQueueItemData(ctx context.Context, id bson.ObjectID, item ports.QueueItem) error {
	if err := q.repository.UpdateData(ctx, id, item.GetData()); err != nil {
		return errors.Wrap(err, "failed to update queue item data")
	}

	return nil
}

func (q Service) UpdateQueueItemStatus(ctx context.Context, id bson.ObjectID, status ports.QueueItemStatus) error {
	if err := q.repository.UpdateStatus(ctx, id, status, ""); err != nil {
		return errors.Wrap(err, "failed to update queue item status")
	}

	return nil
}

func (q Service) Filter(
	ctx context.Context,
	action ports.ActionType,
	status ports.QueueItemStatus,
) ([]ports.QueueItem, error) {
	items, err := q.repository.FindByStatusAndAction(ctx, q.tag, status, action)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find queue items")
	}

	transformed, err := q.transformAll(items)
	if err != nil {
		return nil, errors.Wrap(err, "failed to transform queue items")
	}

	return transformed, nil
}

func (q Service) Filters(
	ctx context.Context,
	actions []ports.ActionType,
	status ports.QueueItemStatus,
) ([]ports.QueueItem, error) {
	items, err := q.repository.FindByStatusAndActions(ctx, q.tag, status, actions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find queue items")
	}

	transformed, err := q.transformAll(items)
	if err != nil {
		return nil, errors.Wrap(err, "failed to transform queue items")
	}

	return transformed, nil
}

func (q Service) transformAll(items []ports.QueueModel) ([]ports.QueueItem, error) {
	if len(items) == 0 {
		return nil, nil
	}

	transformed := make([]ports.QueueItem, 0, len(items))
	for _, item := range items {
		t, err := q.transformer.Transform(item)
		if err != nil {
			return nil, errors.Wrap(err, "failed to transform queue item")
		}

		transformed = append(transformed, t)
	}

	return transformed, nil
}

func (q Service) IsEmpty(ctx context.Context) bool {
	count, err := q.repository.CountByStatuses(
		ctx,
		q.tag,
		[]ports.QueueItemStatus{ports.ItemStatusPending, ports.ItemStatusRetry},
	)
	if err != nil {
		return true
	}

	return count == 0
}

func (q Service) GetStatus(ctx context.Context, id bson.ObjectID) (ports.QueueItemStatus, error) {
	item, err := q.repository.FindByID(ctx, id)
	if err != nil {
		return ports.ItemStatusUnknown, errors.Wrap(err, "failed to get queue item status")
	}

	return item.Status, nil
}

func (q Service) GetStatusWithMessage(ctx context.Context, id bson.ObjectID) (ports.QueueItemStatus, string, error) {
	item, err := q.repository.FindByID(ctx, id)
	if err != nil {
		return ports.ItemStatusUnknown, "", errors.Wrap(err, "failed to get queue item status")
	}

	return item.Status, item.Message, nil
}

func (q Service) GetItem(ctx context.Context, id bson.ObjectID) (ports.QueueItem, error) {
	item, err := q.repository.FindByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get queue item")
	}

	transformed, err := q.transformer.Transform(item)
	if err != nil {
		return nil, errors.Wrap(err, "failed to transform queue item")
	}

	return transformed, nil
}

func (q Service) GetQueueData(ctx context.Context, id bson.ObjectID) (ports.QueueItem, error) {
	item, err := q.repository.FindByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get queue item")
	}

	transformed, err := q.transformer.Transform(item)
	if err != nil {
		return nil, errors.Wrap(err, "failed to transform queue item")
	}

	return ports.NewWrappedItemWithStatus(transformed, item.Status, item.Message), nil
}

func (q Service) ListItems(
	ctx context.Context,
	filter ports.ListQueueItemsFilter,
) ([]ports.QueueItem, error) {
	if filter.Tag == "" {
		filter.Tag = q.tag
	}

	items, err := q.repository.ListItems(ctx, filter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list queue items")
	}

	transformed := make([]ports.QueueItem, 0, len(items))
	for _, m := range items {
		item, err := q.transformer.Transform(m)
		if err != nil {
			q.logger().Error().Err(err).Str("id", m.Id.Hex()).Msg("failed to transform queue item in list")
			continue
		}

		transformed = append(
			transformed,
			ports.NewWrappedItemWithStatus(item, m.Status, m.Message),
		)
	}

	return transformed, nil
}
