package example

import (
	"context"
	"time"

	"{{MODULE_PATH}}/internal/ports"

	hexports "github.com/king-glitch/hexag/framework/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Service struct {
	hexports.ServiceBase
	repository ports.ExampleRepository
}

func NewService(ctx ports.ServiceContext, repository ports.ExampleRepository) ports.ExampleService {
	return Service{
		ServiceBase: hexports.NewBaseService(ctx),
		repository:  repository,
	}
}

func (s Service) Get(ctx context.Context, id bson.ObjectID) (ports.ExampleModel, *hexports.ServiceError) {
	example, err := s.repository.Get(ctx, id)
	if err != nil {
		return ports.ExampleModel{}, hexports.NewServiceErrorFromCause(err, "failed to get example")
	}

	return example, nil
}

func (s Service) Create(ctx context.Context, name string, at time.Time) (ports.ExampleModel, *hexports.ServiceError) {
	id, err := s.repository.Create(ctx, ports.ExampleModel{Name: name}, at)
	if err != nil {
		return ports.ExampleModel{}, hexports.NewServiceErrorFromCause(err, "failed to create example")
	}

	created, err := s.repository.Get(ctx, id)
	if err != nil {
		return ports.ExampleModel{}, hexports.NewServiceErrorFromCause(err, "failed to load created example")
	}

	return created, nil
}
