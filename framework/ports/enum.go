package ports

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
