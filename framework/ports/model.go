package ports

import (
	"encoding/json"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Model interface {
	CollectionName() string
}

type BaseSetter[T any] interface {
	WithBase(ModelBase) T
}

type WithUserID struct {
	UserID bson.ObjectID `json:"user_id" bson:"user_id"`
}

type ModelBase struct {
	ID        bson.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	CreatedAt time.Time     `json:"created_at" bson:"created_at,omitempty"`
	UpdatedAt time.Time     `json:"updated_at" bson:"updated_at,omitempty"`
}

func (m ModelBase) GetID() bson.ObjectID {
	return m.ID
}

func (m ModelBase) GetCreatedAt() time.Time {
	return m.CreatedAt
}

func (m ModelBase) GetUpdatedAt() time.Time {
	return m.UpdatedAt
}

func (m ModelBase) WithBase(base ModelBase) ModelBase {
	m.ID = base.ID
	m.CreatedAt = base.CreatedAt
	m.UpdatedAt = base.UpdatedAt

	return m
}

// MarshalOmitBase implements the shared MarshalJSON body every generated
// model uses: it hides id/created_at/updated_at while any of them is still
// zero (i.e. the model was never persisted), so a not-yet-created row never
// serializes a fake zero ObjectID or zero timestamp to a client.
func MarshalOmitBase(base ModelBase, v any) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	if !base.ID.IsZero() && !base.CreatedAt.IsZero() && !base.UpdatedAt.IsZero() {
		return data, nil
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}

	if base.ID.IsZero() {
		delete(fields, "id")
	}
	if base.CreatedAt.IsZero() {
		delete(fields, "created_at")
	}
	if base.UpdatedAt.IsZero() {
		delete(fields, "updated_at")
	}

	return json.Marshal(fields)
}
