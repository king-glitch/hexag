package ports

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type QueueModel struct {
	Id         bson.ObjectID   `bson:"_id" json:"id"`
	Data       any             `bson:"data" json:"data"`
	Tag        string          `bson:"tag" json:"tag"`
	ActionType ActionType      `bson:"action_type" json:"action_type"`
	Message    string          `bson:"message" json:"message"`
	Status     QueueItemStatus `bson:"status" json:"status"`
	Priority   Priority        `bson:"priority" json:"priority"`

	ScheduledAt *time.Time `bson:"scheduled_at" json:"scheduled_at"`
	Attempts    int        `bson:"attempts" json:"attempts"`
	CreatedAt   time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `bson:"updated_at" json:"updated_at"`
}

func (m QueueModel) CollectionName() string {
	return "queue"
}

type QueueItemStatus string

const (
	ItemStatusUnknown   QueueItemStatus = "unknown"
	ItemStatusPending   QueueItemStatus = "pending"
	ItemStatusDequeued  QueueItemStatus = "dequeued"
	ItemStatusCompleted QueueItemStatus = "completed"
	ItemStatusFailed    QueueItemStatus = "failed"
	ItemStatusCanceled  QueueItemStatus = "canceled"
	ItemStatusRetry     QueueItemStatus = "retry"
)

type Priority int

const (
	PriorityLowest  Priority = 0
	PriorityLow     Priority = 1
	PriorityNormal  Priority = 2
	PriorityHigh    Priority = 3
	PriorityHighest Priority = 4
)

type ActionType string

func (actionType ActionType) String() string {
	return string(actionType)
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

type Item struct {
	Id          bson.ObjectID   `json:"id" bson:"_id"`
	Data        any             `json:"data" bson:"data"`
	ActionType  ActionType      `json:"action_type" bson:"action_type"`
	Priority    Priority        `json:"priority" bson:"priority"`
	ScheduledAt *time.Time      `json:"scheduled_at,omitempty" bson:"scheduled_at,omitempty"`
	Status      QueueItemStatus `json:"status,omitempty" bson:"status,omitempty"`
	Message     string          `json:"message,omitempty" bson:"message,omitempty"`
}

func NewItem[T any](actionType ActionType, data T, scheduledAt ...time.Time) QueueItem {
	var s *time.Time
	if len(scheduledAt) > 0 {
		s = &scheduledAt[0]
	}

	return &Item{
		Data:        data,
		ActionType:  actionType,
		Priority:    PriorityNormal,
		ScheduledAt: s,
	}
}

func NewItemWithPriority[T any](
	actionType ActionType,
	data T,
	priority Priority,
	scheduledAt ...time.Time,
) QueueItem {
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

func (i *Item) GetActionType() ActionType {
	return i.ActionType
}

func (i *Item) GetPriority() Priority {
	return i.Priority
}

func (i *Item) GetScheduledAt() *time.Time {
	return i.ScheduledAt
}

func (i *Item) GetStatus() QueueItemStatus {
	return i.Status
}

func (i *Item) GetMessage() string {
	return i.Message
}

func NewWrappedItemWithStatus(
	data QueueItem,
	status QueueItemStatus,
	message string,
) QueueItem {
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

type Transformer interface {
	Transform(QueueModel) (QueueItem, error)
}

type QueueExecutor interface {
	Run(terminatedChan <-chan bool, stoppedChan chan<- bool) error
}

type QueueRepository interface {
	InsertItem(ctx context.Context, item QueueModel) (bson.ObjectID, error)
	ClaimReadyItems(ctx context.Context, tag string, at time.Time, limit int) ([]QueueModel, error)
	FindReadyItems(ctx context.Context, tag string, at time.Time, limit int) ([]QueueModel, error)
	FindByID(ctx context.Context, id bson.ObjectID) (QueueModel, error)
	FindByStatusAndAction(
		ctx context.Context,
		tag string,
		status QueueItemStatus,
		action ActionType,
	) ([]QueueModel, error)
	FindByStatusAndActions(
		ctx context.Context,
		tag string,
		status QueueItemStatus,
		actions []ActionType,
	) ([]QueueModel, error)

	BulkUpdateStatus(ctx context.Context, updates []QueueItemStatusUpdate) error
	UpdateStatus(ctx context.Context, id bson.ObjectID, status QueueItemStatus, message string) error
	RescheduleItem(ctx context.Context, id bson.ObjectID, scheduledAt time.Time, message string) error
	UpdateManyStatus(ctx context.Context, ids []bson.ObjectID, status QueueItemStatus, message string) error
	UpdateData(ctx context.Context, id bson.ObjectID, data any) error

	CountByStatuses(ctx context.Context, tag string, statuses []QueueItemStatus) (int64, error)
	ListItems(ctx context.Context, filter ListQueueItemsFilter) ([]QueueModel, error)
}

type QueueService interface {
	GetTag() string
	Enqueue(ctx context.Context, item QueueItem) (bson.ObjectID, error)
	Dequeue(ctx context.Context, fn func(QueueItem) error, options ...DequeueOption) error
	DequeueWithManualAck(
		ctx context.Context,
		item func(QueueItem) (QueueItemStatus, error),
		options ...DequeueOption,
	) error

	Cancel(ctx context.Context, id bson.ObjectID, reason ...string) error
	BulkCancel(ctx context.Context, ids []bson.ObjectID, reason ...string) error
	UpdateQueueItemData(ctx context.Context, id bson.ObjectID, item QueueItem) error
	UpdateQueueItemStatus(ctx context.Context, id bson.ObjectID, status QueueItemStatus) error

	Filter(ctx context.Context, action ActionType, status QueueItemStatus) ([]QueueItem, error)
	Filters(ctx context.Context, actions []ActionType, status QueueItemStatus) ([]QueueItem, error)

	IsEmpty(ctx context.Context) bool
	GetStatus(ctx context.Context, id bson.ObjectID) (QueueItemStatus, error)
	GetStatusWithMessage(ctx context.Context, id bson.ObjectID) (QueueItemStatus, string, error)
	GetQueueData(ctx context.Context, id bson.ObjectID) (QueueItem, error)
	GetItem(ctx context.Context, id bson.ObjectID) (QueueItem, error)
	ListItems(ctx context.Context, filter ListQueueItemsFilter) ([]QueueItem, error)
}
