package mongo

import (
	"time"

	"github.com/king-glitch/hexag/framework/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func GenerateBaseModel[T ports.BaseSetter[T]](model T, at time.Time) T {
	base := ports.ModelBase{
		ID:        bson.NewObjectID(),
		CreatedAt: at,
		UpdatedAt: at,
	}

	return model.WithBase(base)
}
