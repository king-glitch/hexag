package ports

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Model interface {
	CollectionName() string
}

type ModelBase interface {
	GetID() bson.ObjectID
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
}

type BaseSetter[T any] interface {
	WithBase(ModelBase) T
}

type Validatable interface {
	IsValid() bool
}
