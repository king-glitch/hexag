package ports

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type QueueModel struct {
	Id          bson.ObjectID   `bson:"_id" json:"id"`
	Data        any             `bson:"data" json:"data"`
	Tag         string          `bson:"tag" json:"tag"`
	ActionType  ActionType      `bson:"action_type" json:"action_type"`
	Message     string          `bson:"message" json:"message"`
	Status      QueueItemStatus `bson:"status" json:"status"`
	Priority    Priority        `bson:"priority" json:"priority"`

	ScheduledAt *time.Time `bson:"scheduled_at" json:"scheduled_at"`
	Attempts    int        `bson:"attempts" json:"attempts"`
	CreatedAt   time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `bson:"updated_at" json:"updated_at"`
}

func (m QueueModel) CollectionName() string {
	return "queue"
}

func (m QueueModel) GetID() bson.ObjectID {
	return m.Id
}

func (m QueueModel) GetCreatedAt() time.Time {
	return m.CreatedAt
}

func (m QueueModel) GetUpdatedAt() time.Time {
	return m.UpdatedAt
}

func (m QueueModel) WithBase(base ModelBase) QueueModel {
	if base != nil {
		m.Id = base.GetID()
		m.CreatedAt = base.GetCreatedAt()
		m.UpdatedAt = base.GetUpdatedAt()
	}
	return m
}

type DequeueOptionData struct {
	Limit *int
	At    *time.Time
}

type DequeueOption func(data *DequeueOptionData)

func DequeueOptionWithLimit(limit int) DequeueOption {
	return func(data *DequeueOptionData) {
		data.Limit = &limit
	}
}

func DequeueOptionWithAt(at time.Time) DequeueOption {
	return func(data *DequeueOptionData) {
		data.At = &at
	}
}

type QueueItemStatusUpdate struct {
	ID      bson.ObjectID
	Status  QueueItemStatus
	Message string
}

type ListQueueItemsFilter struct {
	UserID     *bson.ObjectID
	Tag        string
	ActionType ActionType
	Statuses   []QueueItemStatus
	Limit      int
}

type QueueItem interface {
	GetId() bson.ObjectID
	GetData() any
	SetData(data any)
	GetActionType() ActionType
	GetPriority() Priority
	GetScheduledAt() *time.Time
	GetStatus() QueueItemStatus
	GetMessage() string
}

type Transformer interface {
	Transform(QueueModel) (QueueItem, error)
}

type QueueExecutor interface {
	Run(terminatedChan <-chan bool, stoppedChan chan<- bool) error
}
