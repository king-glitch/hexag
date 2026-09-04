package queue

import (
	"context"
	"testing"

	"github.com/king-glitch/hexag/framework/ports"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type stubContext struct{}

func (stubContext) Logger() *zerolog.Logger {
	logger := zerolog.Nop()
	return &logger
}

// stubQueue embeds the interface so only the methods the executor touches
// need implementing; anything else panics loudly if it is ever called.
type stubQueue struct {
	ports.QueueService
}

func (stubQueue) GetTag() string { return "test" }

func newTestExecutor(options ...ExecutorOption) *Executor {
	return NewExecutor(stubContext{}, stubQueue{}, options...)
}

func TestExecutorProcess(t *testing.T) {
	const action ports.ActionType = "test.action"

	boom := errors.New("boom")

	tests := []struct {
		name       string
		register   bool
		handlerErr error
		retryable  bool
		wantStatus ports.QueueItemStatus
		wantErr    bool
	}{
		{
			name:       "completes when the handler succeeds",
			register:   true,
			retryable:  true,
			wantStatus: ports.ItemStatusCompleted,
		},
		{
			name:       "reschedules a retryable handler error",
			register:   true,
			handlerErr: boom,
			retryable:  true,
			wantStatus: ports.ItemStatusRetry,
		},
		{
			name:       "fails terminally when the error is not retryable",
			register:   true,
			handlerErr: boom,
			retryable:  false,
			wantErr:    true,
		},
		{
			name:      "fails terminally when no handler is registered",
			register:  false,
			retryable: true,
			wantErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			executor := newTestExecutor(
				WithRetryPredicate(func(error) bool { return test.retryable }),
			)

			if test.register {
				executor.Register(
					action,
					func(context.Context, ports.QueueItem) error { return test.handlerErr },
				)
			}

			status, err := executor.process(
				context.Background(),
				ports.NewItem(action, map[string]string{"k": "v"}),
			)

			if test.wantErr {
				assert.Error(t, err)
				assert.Empty(t, status)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, test.wantStatus, status)
		})
	}
}

func TestExecutorRunStopsOnContextCancel(t *testing.T) {
	executor := newTestExecutor()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	assert.NoError(t, executor.Run(ctx))
}
