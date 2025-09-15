package ranked

import (
	"github.com/ddirect/container"
	"github.com/ddirect/container/internal/rankedlist"
)

type MapItem[K comparable, R container.Comparer[R], V any] rankedlist.Item[R, V, K]

type MutableMapItem[K comparable, R container.Comparer[R], V any] struct {
	*MapItem[K, R, V]
	parent *Map[K, R, V]
}

func (m *Map[K, R, V]) mutableMapItem(item *rankedItem[K, R, V]) MutableMapItem[K, R, V] {
	return MutableMapItem[K, R, V]{mapItem(item), m}
}

func mapItem[K comparable, R container.Comparer[R], V any](item *rankedItem[K, R, V]) *MapItem[K, R, V] {
	return (*MapItem[K, R, V])(item)
}

func listItem[K comparable, R container.Comparer[R], V any](it *MapItem[K, R, V]) *rankedlist.Item[R, V, K] {
	return (*rankedlist.Item[R, V, K])(it)
}

func (it *MapItem[K, R, V]) Present() bool {
	return listItem(it).Present()
}

func (it *MapItem[K, R, V]) Key() K {
	return listItem(it).Auxiliary()
}

func (it *MapItem[K, R, V]) Rank() R {
	return listItem(it).Rank()
}

func (it MutableMapItem[K, R, V]) SetRank(rank R) {
	it.parent.r.SetRank(listItem(it.MapItem), rank)
}

func (it MutableMapItem[K, R, V]) Delete() {
	it.parent.deleteItem(listItem(it.MapItem))
}
