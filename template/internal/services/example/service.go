package example

import (
	"context"
	"time"

	"{{MODULE_PATH}}/internal/ports"

	servicebase "github.com/king-glitch/hexag/framework/api/service/base"
	serviceerrors "github.com/king-glitch/hexag/framework/api/service/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Service struct {
	servicebase.ServiceBase
	repository ports.ExampleRepository
}

func NewService(ctx ports.ServiceContext, repository ports.ExampleRepository) ports.ExampleService {
	return Service{
		ServiceBase: servicebase.NewBaseService(ctx),
		repository:  repository,
	}
}

func (s Service) Get(ctx context.Context, id bson.ObjectID) (ports.ExampleModel, *serviceerrors.ServiceError) {
	example, err := s.repository.Get(ctx, id)
	if err != nil {
		return ports.ExampleModel{}, serviceerrors.NewServiceErrorFromCause(err, "failed to get example")
	}

	return example, nil
}

func (s Service) Create(ctx context.Context, name string, at time.Time) (ports.ExampleModel, *serviceerrors.ServiceError) {
	id, err := s.repository.Create(ctx, ports.ExampleModel{Name: name}, at)
	if err != nil {
		return ports.ExampleModel{}, serviceerrors.NewServiceErrorFromCause(err, "failed to create example")
	}

	created, err := s.repository.Get(ctx, id)
	if err != nil {
		return ports.ExampleModel{}, serviceerrors.NewServiceErrorFromCause(err, "failed to load created example")
	}

	return created, nil
}
