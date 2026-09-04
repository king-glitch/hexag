package ports

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type ExampleRepository interface {
	Create(ctx context.Context, example ExampleModel, at time.Time) (bson.ObjectID, error)
	Get(ctx context.Context, id bson.ObjectID) (ExampleModel, error)
}
