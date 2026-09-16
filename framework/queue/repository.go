package queue

import (
	"context"
	"time"

	"github.com/king-glitch/hexag/framework/ports"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var readySort = bson.D{
	Queue.ScheduledAt.Asc(),
	Queue.Priority.Desc(),
	Queue.Id.Asc(),
}

var claimSort = bson.D{
	Queue.Priority.Desc(),
	Queue.CreatedAt.Asc(),
	Queue.Id.Asc(),
}

type Repository struct {
	collection *mongo.Collection
}

func NewRepository(db *mongo.Database) ports.QueueRepository {
	return Repository{collection: db.Collection(ports.QueueModel{}.CollectionName())}
}

func (r Repository) InsertItem(ctx context.Context, item ports.QueueModel) (bson.ObjectID, error) {
	result, err := r.collection.InsertOne(ctx, item)
	if err != nil {
		return bson.ObjectID{}, errors.Wrap(err, "failed to insert queue item")
	}

	return result.InsertedID.(bson.ObjectID), nil
}

func readyFilter(tag string, at time.Time) bson.D {
	return bson.D{
		Queue.Tag.Eq(tag),
		Queue.Status.In(ports.ItemStatusPending, ports.ItemStatusRetry),
		{
			Key: "$or",
			Value: bson.A{
				bson.D{Queue.ScheduledAt.Eq(nil)},
				bson.D{Queue.ScheduledAt.Lte(&at)},
			},
		},
	}
}

func (r Repository) ClaimReadyItems(
	ctx context.Context,
	tag string,
	at time.Time,
	limit int,
) ([]ports.QueueModel, error) {
	if limit <= 0 {
		return nil, nil
	}

	filter := readyFilter(tag, at)
	update := bson.D{
		{
			Key: "$set",
			Value: bson.D{
				Queue.Status.Eq(ports.ItemStatusDequeued),
				Queue.UpdatedAt.Eq(at),
			},
		},
		{
			Key: "$inc",
			Value: bson.D{
				Queue.Attempts.Eq(1),
			},
		},
	}

	var items []ports.QueueModel
	for len(items) < limit {
		result := r.collection.FindOneAndUpdate(
			ctx,
			filter,
			update,
			options.FindOneAndUpdate().SetSort(claimSort).SetReturnDocument(options.After),
		)

		var item ports.QueueModel
		if err := result.Decode(&item); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				break
			}

			return nil, errors.Wrap(err, "failed to claim ready queue item")
		}

		items = append(items, item)
	}

	return items, nil
}

func (r Repository) FindReadyItems(
	ctx context.Context,
	tag string,
	at time.Time,
	limit int,
) ([]ports.QueueModel, error) {
	cursor, err := r.collection.Find(
		ctx,
		readyFilter(tag, at),
		options.Find().SetSort(readySort).SetLimit(int64(limit)),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find ready queue items")
	}
	defer func() { _ = cursor.Close(ctx) }()

	var items []ports.QueueModel
	if err := cursor.All(ctx, &items); err != nil {
		return nil, errors.Wrap(err, "failed to decode ready queue items")
	}

	return items, nil
}

func (r Repository) FindByID(ctx context.Context, id bson.ObjectID) (ports.QueueModel, error) {
	var item ports.QueueModel
	if err := r.collection.FindOne(ctx, bson.D{Queue.Id.Eq(id)}).Decode(&item); err != nil {
		return ports.QueueModel{}, errors.Wrap(err, "failed to find queue item")
	}

	return item, nil
}

func (r Repository) FindByStatusAndAction(
	ctx context.Context,
	tag string,
	status ports.QueueItemStatus,
	action ports.ActionType,
) ([]ports.QueueModel, error) {
	filter := bson.D{
		Queue.Tag.Eq(tag),
		Queue.Status.Eq(status),
		Queue.ActionType.Eq(action),
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(readySort))
	if err != nil {
		return nil, errors.Wrap(err, "failed to find queue items")
	}
	defer func() { _ = cursor.Close(ctx) }()

	var items []ports.QueueModel
	if err := cursor.All(ctx, &items); err != nil {
		return nil, errors.Wrap(err, "failed to decode queue items")
	}

	return items, nil
}

func (r Repository) FindByStatusAndActions(
	ctx context.Context,
	tag string,
	status ports.QueueItemStatus,
	actions []ports.ActionType,
) ([]ports.QueueModel, error) {
	filter := bson.D{
		Queue.Tag.Eq(tag),
		Queue.Status.Eq(status),
		Queue.ActionType.In(actions...),
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(readySort))
	if err != nil {
		return nil, errors.Wrap(err, "failed to find queue items")
	}
	defer func() { _ = cursor.Close(ctx) }()

	var items []ports.QueueModel
	if err := cursor.All(ctx, &items); err != nil {
		return nil, errors.Wrap(err, "failed to decode queue items")
	}

	return items, nil
}

func statusUpdateDoc(status ports.QueueItemStatus, message string) bson.D {
	set := bson.D{
		Queue.Status.Eq(status),
		Queue.UpdatedAt.Eq(time.Now()),
	}
	if message != "" {
		set = append(set, Queue.Message.Eq(message))
	}

	return bson.D{{Key: "$set", Value: set}}
}

func (r Repository) BulkUpdateStatus(ctx context.Context, updates []ports.QueueItemStatusUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	writes := make([]mongo.WriteModel, 0, len(updates))
	for _, update := range updates {
		writes = append(
			writes,
			mongo.NewUpdateOneModel().
				SetFilter(bson.D{Queue.Id.Eq(update.ID)}).
				SetUpdate(statusUpdateDoc(update.Status, update.Message)),
		)
	}

	if _, err := r.collection.BulkWrite(ctx, writes); err != nil {
		return errors.Wrap(err, "failed to bulk update queue items")
	}

	return nil
}

func (r Repository) UpdateStatus(
	ctx context.Context,
	id bson.ObjectID,
	status ports.QueueItemStatus,
	message string,
) error {
	if _, err := r.collection.UpdateOne(
		ctx,
		bson.D{Queue.Id.Eq(id)},
		statusUpdateDoc(status, message),
	); err != nil {
		return errors.Wrap(err, "failed to update queue item status")
	}

	return nil
}

func (r Repository) RescheduleItem(
	ctx context.Context,
	id bson.ObjectID,
	scheduledAt time.Time,
	message string,
) error {
	set := bson.D{
		Queue.Status.Eq(ports.ItemStatusRetry),
		Queue.ScheduledAt.Eq(&scheduledAt),
		Queue.UpdatedAt.Eq(time.Now()),
	}
	if message != "" {
		set = append(set, Queue.Message.Eq(message))
	}

	if _, err := r.collection.UpdateOne(
		ctx,
		bson.D{Queue.Id.Eq(id)},
		bson.D{{Key: "$set", Value: set}},
	); err != nil {
		return errors.Wrap(err, "failed to reschedule queue item")
	}

	return nil
}

func (r Repository) UpdateManyStatus(
	ctx context.Context,
	ids []bson.ObjectID,
	status ports.QueueItemStatus,
	message string,
) error {
	if _, err := r.collection.UpdateMany(
		ctx,
		bson.D{Queue.Id.In(ids...)},
		statusUpdateDoc(status, message),
	); err != nil {
		return errors.Wrap(err, "failed to bulk update queue item status")
	}

	return nil
}

func (r Repository) UpdateData(ctx context.Context, id bson.ObjectID, data any) error {
	if _, err := r.collection.UpdateOne(
		ctx,
		bson.D{Queue.Id.Eq(id)},
		bson.D{
			{
				Key: "$set",
				Value: bson.D{
					Queue.Data.Eq(data),
					Queue.UpdatedAt.Eq(time.Now()),
				},
			},
		},
	); err != nil {
		return errors.Wrap(err, "failed to update queue item data")
	}

	return nil
}

func (r Repository) CountByStatuses(
	ctx context.Context,
	tag string,
	statuses []ports.QueueItemStatus,
) (int64, error) {
	count, err := r.collection.CountDocuments(
		ctx, bson.D{
			Queue.Tag.Eq(tag),
			Queue.Status.In(statuses...),
		},
	)
	if err != nil {
		return 0, errors.Wrap(err, "failed to count queue items")
	}

	return count, nil
}

func (r Repository) ListItems(
	ctx context.Context,
	filter ports.ListQueueItemsFilter,
) ([]ports.QueueModel, error) {
	query := bson.D{}

	if filter.Tag != "" {
		query = append(query, Queue.Tag.Eq(filter.Tag))
	}

	if filter.ActionType != "" {
		query = append(query, Queue.ActionType.Eq(filter.ActionType))
	}

	if len(filter.Statuses) > 0 {
		query = append(query, Queue.Status.In(filter.Statuses...))
	}

	if filter.UserID != nil {
		query = append(query, bson.E{Key: "data.user_id", Value: *filter.UserID})
	}

	limit := int64(50)
	if filter.Limit > 0 {
		limit = int64(filter.Limit)
	}

	cursor, err := r.collection.Find(
		ctx,
		query,
		options.Find().
			SetProjection(bson.D{{Key: "data.image", Value: 0}}).
			SetSort(bson.D{Queue.Id.Desc()}).
			SetAllowDiskUse(true).
			SetLimit(limit),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list queue items")
	}
	defer func() { _ = cursor.Close(ctx) }()

	var items []ports.QueueModel
	if err := cursor.All(ctx, &items); err != nil {
		return nil, errors.Wrap(err, "failed to decode queue items")
	}

	return items, nil
}
