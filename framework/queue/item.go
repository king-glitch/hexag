package queue

import (
	"time"

	"github.com/king-glitch/hexag/framework/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Item struct {
	Id          bson.ObjectID         `json:"id" bson:"_id"`
	Data        any                   `json:"data" bson:"data"`
	ActionType  ports.ActionType      `json:"action_type" bson:"action_type"`
	Priority    ports.Priority        `json:"priority" bson:"priority"`
	ScheduledAt *time.Time            `json:"scheduled_at,omitempty" bson:"scheduled_at,omitempty"`
	Status      ports.QueueItemStatus `json:"status,omitempty" bson:"status,omitempty"`
	Message     string                `json:"message,omitempty" bson:"message,omitempty"`
}

func NewItem[T any](actionType ports.ActionType, data T, scheduledAt ...time.Time) ports.QueueItem {
	var s *time.Time
	if len(scheduledAt) > 0 {
		s = &scheduledAt[0]
	}

	return &Item{
		Data:        data,
		ActionType:  actionType,
		Priority:    ports.PriorityNormal,
		ScheduledAt: s,
	}
}

func NewItemWithPriority[T any](
	actionType ports.ActionType,
	data T,
	priority ports.Priority,
	scheduledAt ...time.Time,
) ports.QueueItem {
	var s *time.Time
	if len(scheduledAt) > 0 {
		s = &scheduledAt[0]
	}

	return &Item{
		Data:        data,
		ActionType:  actionType,
		Priority:    priority,
		ScheduledAt: s,
	}
}

func (i *Item) GetId() bson.ObjectID {
	return i.Id
}

func (i *Item) GetData() any {
	return i.Data
}

func (i *Item) SetData(data any) {
	i.Data = data
}

func (i *Item) GetActionType() ports.ActionType {
	return i.ActionType
}

func (i *Item) GetPriority() ports.Priority {
	return i.Priority
}

func (i *Item) GetScheduledAt() *time.Time {
	return i.ScheduledAt
}

func (i *Item) GetStatus() ports.QueueItemStatus {
	return i.Status
}

func (i *Item) GetMessage() string {
	return i.Message
}

func NewWrappedItemWithStatus(
	data ports.QueueItem,
	status ports.QueueItemStatus,
	message string,
) ports.QueueItem {
	return &Item{
		Id:          data.GetId(),
		Data:        data.GetData(),
		ActionType:  data.GetActionType(),
		Priority:    data.GetPriority(),
		ScheduledAt: data.GetScheduledAt(),
		Status:      status,
		Message:     message,
	}
}
