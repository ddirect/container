package rankedlist

import (
	"github.com/ddirect/container"
)

type Item[R container.Comparer[R], T any] struct {
	value T
	rank  R
	idx   int // index = idx + seed
}

func (it *Item[R, T]) Present() bool {
	// due to the high value of seed, all idx values resulting in valid indexes are negative
	return it != nil && it.idx < 0
}

func (it *Item[R, T]) setNotPresent() {
	it.idx = 0
}

func (it *Item[R, T]) Value() *T {
	return &it.value
}

func (it *Item[R, T]) Rank() R {
	return it.rank
}
