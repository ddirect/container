package rankedlist

import (
	"github.com/ddirect/container"
)

type Item[R container.Comparer[R], T any, A any] struct {
	Value   T
	rank    R
	aux     A
	indexP1 uint // index plus 1 - if zero, the item does not belong to the container
}

func (it *Item[R, T, A]) Rank() R {
	return it.rank
}

func (it *Item[R, T, A]) Auxiliary() A {
	return it.aux
}

func (it *Item[R, T, A]) Present() bool {
	return it != nil && it.indexP1 > 0
}

func (it *Item[R, T, A]) setNotPresent() {
	it.indexP1 = 0
}
