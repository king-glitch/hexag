package ports

import "math"

type PaginationParams struct {
	Page   int
	Amount int

	Queries map[string]string
	Sorts   map[string]string
}

func (p PaginationParams) SafePage() int {
	if p.Page < 1 {
		return 1
	}
	return p.Page
}

func (p PaginationParams) SafeAmount() int {
	if p.Amount < 1 {
		return 10
	}
	return p.Amount
}

func (p PaginationParams) Offset() int {
	return (p.SafePage() - 1) * p.SafeAmount()
}

type CollectionMeta struct {
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type Collection[T any] struct {
	Collection []T `json:"collection"`

	Count int64          `json:"count"`
	Meta  CollectionMeta `json:"meta"`
}

func NewCollection[T any](collection []T, total int64, perPage int) Collection[T] {
	if collection == nil {
		collection = make([]T, 0)
	}

	return Collection[T]{
		Collection: collection,
		Count:      int64(len(collection)),
		Meta: CollectionMeta{
			Total:      total,
			TotalPages: int(math.Ceil(float64(total) / float64(perPage))),
		},
	}
}

func NewEmptyCollection[T any]() Collection[T] {
	return Collection[T]{
		Collection: make([]T, 0),
		Count:      0,
		Meta: CollectionMeta{
			Total:      0,
			TotalPages: 0,
		},
	}
}
