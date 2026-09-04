package example

import (
	"context"
	"time"

	"{{MODULE_PATH}}/internal/adapters/database/mongo/models"
	"{{MODULE_PATH}}/internal/ports"

	hexmongo "github.com/king-glitch/hexag/framework/mongo"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Repository struct {
	collection *mongo.Collection
}

func NewRepository(db *mongo.Database) ports.ExampleRepository {
	return Repository{collection: db.Collection(ports.ExampleModel{}.CollectionName())}
}

func (r Repository) Create(ctx context.Context, example ports.ExampleModel, at time.Time) (bson.ObjectID, error) {
	model := hexmongo.GenerateBaseModel(example, at)

	if _, err := r.collection.InsertOne(ctx, model); err != nil {
		return bson.ObjectID{}, errors.Wrap(err, "failed to insert example")
	}

	return model.ID, nil
}

func (r Repository) Get(ctx context.Context, id bson.ObjectID) (ports.ExampleModel, error) {
	var example ports.ExampleModel
	if err := r.collection.FindOne(ctx, bson.D{models.Example.ID.Eq(id)}).Decode(&example); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ports.ExampleModel{}, ports.ErrExampleNotFound
		}

		return ports.ExampleModel{}, errors.Wrap(err, "failed to find example")
	}

	return example, nil
}
