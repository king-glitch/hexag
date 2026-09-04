package queue

import (
	"context"
	"time"

	"github.com/king-glitch/hexag/framework/ports"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

const defaultPollInterval = 5 * time.Second

// Handler processes one dequeued item. Returning an error surrenders the
// terminal-vs-retry decision to the executor's retry predicate.
type Handler func(ctx context.Context, item ports.QueueItem) error

// RetryPredicate decides whether a handler error is worth another attempt.
// Projects supply their own so the framework never learns project sentinels.
type RetryPredicate func(err error) bool

type ExecutorOption func(*Executor)

func WithPollInterval(interval time.Duration) ExecutorOption {
	return func(e *Executor) { e.interval = interval }
}

// WithRetryPredicate overrides the default of retrying every handler error.
func WithRetryPredicate(predicate RetryPredicate) ExecutorOption {
	return func(e *Executor) { e.retryable = predicate }
}

type Executor struct {
	service   ports.QueueService
	logger    *zerolog.Logger
	interval  time.Duration
	handlers  map[ports.ActionType]Handler
	retryable RetryPredicate
}

func NewExecutor(
	serviceContext ports.ServiceContext,
	service ports.QueueService,
	options ...ExecutorOption,
) *Executor {
	logger := serviceContext.Logger()
	if logger == nil {
		nop := zerolog.Nop()
		logger = &nop
	}

	e := &Executor{
		service:   service,
		logger:    logger,
		interval:  defaultPollInterval,
		handlers:  map[ports.ActionType]Handler{},
		retryable: func(error) bool { return true },
	}

	for _, apply := range options {
		apply(e)
	}

	return e
}

func (e *Executor) Register(actionType ports.ActionType, handler Handler) {
	e.handlers[actionType] = handler
}

// Run polls until ctx is cancelled. A poll error is logged and retried on the
// next tick rather than killing the loop — a transient database blip must not
// take the whole worker down for the life of the process.
func (e *Executor) Run(ctx context.Context) error {
	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()

	e.logger.Info().
		Str("tag", e.service.GetTag()).
		Dur("interval", e.interval).
		Int("handlers", len(e.handlers)).
		Msg("queue executor started")

	for {
		select {
		case <-ctx.Done():
			e.logger.Info().Str("tag", e.service.GetTag()).Msg("queue executor stopped")

			return nil
		case <-ticker.C:
			if err := e.tick(ctx); err != nil {
				e.logger.Error().Err(err).Str("tag", e.service.GetTag()).Msg("queue poll failed")
			}
		}
	}
}

func (e *Executor) tick(ctx context.Context) error {
	if err := e.service.DequeueWithManualAck(
		ctx,
		func(item ports.QueueItem) (ports.QueueItemStatus, error) {
			return e.process(ctx, item)
		},
	); err != nil {
		return errors.Wrap(err, "failed to dequeue queue items")
	}

	return nil
}

// process maps a handler outcome onto the status vocabulary Service.dequeue
// expects: a non-nil error fails the item terminally and keeps the message,
// ItemStatusRetry reschedules it with backoff.
func (e *Executor) process(ctx context.Context, item ports.QueueItem) (ports.QueueItemStatus, error) {
	handler, ok := e.handlers[item.GetActionType()]
	if !ok {
		return "", errors.Errorf("no handler registered for action type %q", item.GetActionType())
	}

	if err := handler(ctx, item); err != nil {
		if e.retryable(err) {
			e.logger.Warn().
				Err(err).
				Str("id", item.GetId().Hex()).
				Str("action_type", item.GetActionType().String()).
				Msg("queue item failed, scheduling retry")

			return ports.ItemStatusRetry, nil
		}

		return "", errors.Wrap(err, "queue handler failed terminally")
	}

	return ports.ItemStatusCompleted, nil
}
