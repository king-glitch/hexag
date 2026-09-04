package ports

import "context"

type TransactionRunner interface {
	Run(ctx context.Context, fn func(ctx context.Context) error) error
}
