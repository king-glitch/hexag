package ports

import (
	"context"
	"time"

	serviceerrors "github.com/king-glitch/hexag/framework/api/service/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type ExampleService interface {
	Get(ctx context.Context, id bson.ObjectID) (ExampleModel, *serviceerrors.ServiceError)
	Create(ctx context.Context, name string, at time.Time) (ExampleModel, *serviceerrors.ServiceError)
}
