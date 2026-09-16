package base

import (
	"encoding/json"
	"time"

	"github.com/king-glitch/hexag/framework/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
)

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

func (m ModelBase) WithBase(base ports.ModelBase) ModelBase {
	if base == nil {
		return m
	}
	m.ID = base.GetID()
	m.CreatedAt = base.GetCreatedAt()
	m.UpdatedAt = base.GetUpdatedAt()

	return m
}

func MarshalOmitBase(base ports.ModelBase, v any) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	if base == nil {
		return data, nil
	}

	id := base.GetID()
	createdAt := base.GetCreatedAt()
	updatedAt := base.GetUpdatedAt()

	if !id.IsZero() && !createdAt.IsZero() && !updatedAt.IsZero() {
		return data, nil
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}

	if id.IsZero() {
		delete(fields, "id")
	}
	if createdAt.IsZero() {
		delete(fields, "created_at")
	}
	if updatedAt.IsZero() {
		delete(fields, "updated_at")
	}

	return json.Marshal(fields)
}
