package mongo

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Field is a type-safe MongoDB query field. Generated model files
// (mongogen output) build one struct-of-Fields per domain model against
// this type instead of re-generating it per project.
type Field[T any] struct {
	key string
}

func NewField[T any](key string) Field[T] {
	return Field[T]{key: key}
}

func (f Field[T]) Key() string {
	return f.key
}

func (f Field[T]) Eq(val T) bson.E {
	return bson.E{Key: f.key, Value: val}
}

func (f Field[T]) Ne(val T) bson.E {
	return bson.E{Key: f.key, Value: bson.D{{Key: "$ne", Value: val}}}
}

func (f Field[T]) Gt(val T) bson.E {
	return bson.E{Key: f.key, Value: bson.D{{Key: "$gt", Value: val}}}
}

func (f Field[T]) Gte(val T) bson.E {
	return bson.E{Key: f.key, Value: bson.D{{Key: "$gte", Value: val}}}
}

func (f Field[T]) Lt(val T) bson.E {
	return bson.E{Key: f.key, Value: bson.D{{Key: "$lt", Value: val}}}
}

func (f Field[T]) Lte(val T) bson.E {
	return bson.E{Key: f.key, Value: bson.D{{Key: "$lte", Value: val}}}
}

func (f Field[T]) In(vals ...T) bson.E {
	return bson.E{Key: f.key, Value: bson.D{{Key: "$in", Value: vals}}}
}

func (f Field[T]) Nin(vals ...T) bson.E {
	return bson.E{Key: f.key, Value: bson.D{{Key: "$nin", Value: vals}}}
}

func (f Field[T]) Exists(exists bool) bson.E {
	return bson.E{Key: f.key, Value: bson.D{{Key: "$exists", Value: exists}}}
}

func (f Field[T]) Asc() bson.E {
	return bson.E{Key: f.key, Value: 1}
}

func (f Field[T]) Desc() bson.E {
	return bson.E{Key: f.key, Value: -1}
}

func (f Field[T]) Unset() bson.E {
	return bson.E{Key: f.key, Value: ""}
}

type StringField = Field[string]
type IntField = Field[int]
type Int64Field = Field[int64]
type BoolField = Field[bool]
type ObjectIDField = Field[bson.ObjectID]
type TimeField = Field[time.Time]
type AnyField = Field[any]
