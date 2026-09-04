package ports

import (
	"context"
	"time"

	hexports "github.com/king-glitch/hexag/framework/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type ExampleService interface {
	Get(ctx context.Context, id bson.ObjectID) (ExampleModel, *hexports.ServiceError)
	Create(ctx context.Context, name string, at time.Time) (ExampleModel, *hexports.ServiceError)
}
