package rankedmap

import (
	"github.com/ddirect/container"
	"github.com/ddirect/container/internal/rankedlist"
)

type kv[K comparable, V any] struct {
	key   K
	value V
}

type rankedList[K comparable, R container.Comparer[R], V any] = rankedlist.List[R, kv[K, V]]
type rankedItem[K comparable, R container.Comparer[R], V any] = rankedlist.Item[R, kv[K, V]]

type MapItem[K comparable, R container.Comparer[R], V any] struct {
	*rankedItem[K, R, V]
}

func mapItem[K comparable, R container.Comparer[R], V any](item *rankedItem[K, R, V]) MapItem[K, R, V] {
	return MapItem[K, R, V]{item}
}

func (it MapItem[K, R, V]) Key() K {
	return it.rankedItem.Value().key
}

func (it MapItem[K, R, V]) Value() *V {
	return &it.rankedItem.Value().value
}
