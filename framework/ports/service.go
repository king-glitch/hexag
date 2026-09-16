package ports

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
)

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
