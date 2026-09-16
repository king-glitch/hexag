package ports

import "context"

type TransactionRunnerAdapter interface {
	Run(ctx context.Context, fn func(ctx context.Context) error) error
}
