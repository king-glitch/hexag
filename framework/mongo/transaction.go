package mongo

import (
	"context"

	"github.com/king-glitch/hexag/framework/ports"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Runner struct {
	client *mongo.Client
}

func NewRunner(client *mongo.Client) ports.TransactionRunner {
	return Runner{client: client}
}

func (r Runner) Run(ctx context.Context, fn func(ctx context.Context) error) error {
	session, err := r.client.StartSession()
	if err != nil {
		return errors.Wrap(err, "failed to start mongo session")
	}
	defer session.EndSession(ctx)

	if _, err := session.WithTransaction(
		ctx,
		func(ctx context.Context) (any, error) {
			if err := fn(ctx); err != nil {
				return nil, err
			}

			return nil, nil
		},
	); err != nil {
		return errors.Wrap(err, "failed to run transaction")
	}

	return nil
}
