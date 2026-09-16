package collection

import "math"

type PaginationParams struct {
	Page    int `query:"page" json:"page"`
	Amount  int `query:"amount" json:"amount"`
	Default int `query:"default" json:"default"`
}

type Meta struct {
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type Collection[T any] struct {
	Collection []T   `json:"collection"`
	Count      int64 `json:"count"`
	Meta       Meta  `json:"meta"`
}

func NewCollection[T any](items []T, total int64, perPage int) Collection[T] {
	if items == nil {
		items = make([]T, 0)
	}

	return Collection[T]{
		Collection: items,
		Count:      int64(len(items)),
		Meta: Meta{
			Total:      total,
			TotalPages: int(math.Ceil(float64(total) / float64(perPage))),
		},
	}
}

func NewEmptyCollection[T any]() Collection[T] {
	return Collection[T]{
		Collection: make([]T, 0),
		Count:      0,
		Meta: Meta{
			Total:      0,
			TotalPages: 0,
		},
	}
}
