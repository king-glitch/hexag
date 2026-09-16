package ports

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

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
