package queue

import (
	"time"

	hexmongo "github.com/king-glitch/hexag/framework/mongo"
	"github.com/king-glitch/hexag/framework/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Queue is the fixed field set for ports.QueueModel. It is hand-written, not
// mongogen output — QueueModel never varies per project, so there is
// nothing for a generator to regenerate here.
var Queue = struct {
	Id          hexmongo.Field[bson.ObjectID]
	Data        hexmongo.Field[any]
	Tag         hexmongo.Field[string]
	ActionType  hexmongo.Field[ports.ActionType]
	Message     hexmongo.Field[string]
	Status      hexmongo.Field[ports.QueueItemStatus]
	Priority    hexmongo.Field[ports.Priority]
	ScheduledAt hexmongo.Field[*time.Time]
	Attempts    hexmongo.Field[int]
	CreatedAt   hexmongo.Field[time.Time]
	UpdatedAt   hexmongo.Field[time.Time]
}{
	Id:          hexmongo.NewField[bson.ObjectID]("_id"),
	Data:        hexmongo.NewField[any]("data"),
	Tag:         hexmongo.NewField[string]("tag"),
	ActionType:  hexmongo.NewField[ports.ActionType]("action_type"),
	Message:     hexmongo.NewField[string]("message"),
	Status:      hexmongo.NewField[ports.QueueItemStatus]("status"),
	Priority:    hexmongo.NewField[ports.Priority]("priority"),
	ScheduledAt: hexmongo.NewField[*time.Time]("scheduled_at"),
	Attempts:    hexmongo.NewField[int]("attempts"),
	CreatedAt:   hexmongo.NewField[time.Time]("created_at"),
	UpdatedAt:   hexmongo.NewField[time.Time]("updated_at"),
}
