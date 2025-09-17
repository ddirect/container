package rankedlist

import (
	"github.com/ddirect/container"
)

type Item[R container.Comparer[R], T any] struct {
	value   T
	rank    R
	indexP1 uint // index plus 1 - if zero, the item does not belong to the container
}

func (it *Item[R, T]) Value() *T {
	return &it.value
}

func (it *Item[R, T]) Rank() R {
	return it.rank
}

func (it *Item[R, T]) Present() bool {
	return it != nil && it.indexP1 > 0
}

func (it *Item[R, T]) setNotPresent() {
	it.indexP1 = 0
}
