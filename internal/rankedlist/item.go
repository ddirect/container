package rankedlist

import "github.com/ddirect/container"

type Item[R container.Comparer[R], T any, A any] struct {
	Value T
	rank  R
	aux   A
	index uint
}

func (it *Item[R, T, A]) Rank() R {
	return it.rank
}

func (it *Item[R, T, A]) Auxiliary() A {
	return it.aux
}
