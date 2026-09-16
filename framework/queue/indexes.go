package queue

import (
	"context"

	"github.com/king-glitch/hexag/framework/ports"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	if _, err := db.Collection(ports.QueueModel{}.CollectionName()).Indexes().CreateMany(
		ctx,
		[]mongo.IndexModel{
			{
				Keys: bson.D{
					Queue.Tag.Asc(),
					Queue.Status.Asc(),
					Queue.ScheduledAt.Asc(),
					Queue.Priority.Desc(),
				},
				Options: options.Index(),
			},
			{
				Keys: bson.D{
					Queue.Tag.Asc(),
					Queue.Status.Asc(),
					Queue.ActionType.Asc(),
				},
				Options: options.Index(),
			},
		},
	); err != nil {
		return errors.Wrap(err, "failed to ensure queue indexes")
	}

	return nil
}
