package ranked

import (
	"iter"
	"math/rand/v2"

	"github.com/ddirect/container"
	"github.com/ddirect/container/internal/rankedlist"
)

type rankedList[K comparable, R container.Comparer[R], V any] = rankedlist.List[R, V, K]
type rankedItem[K comparable, R container.Comparer[R], V any] = rankedlist.Item[R, V, K]

type Map[K comparable, R container.Comparer[R], V any] struct {
	r rankedList[K, R, V]
	m map[K]*rankedItem[K, R, V]
}

func NewMap[K comparable, R container.Comparer[R], V any]() *Map[K, R, V] {
	return &Map[K, R, V]{
		m: make(map[K]*rankedItem[K, R, V]),
	}
}

func (m *Map[K, R, V]) Len() int {
	return m.r.Len()
}

func (m *Map[K, R, V]) Clear() {
	m.r.Clear()
	clear(m.m)
}

func (m *Map[K, R, V]) GetOrCreate(key K, rankIfCreated R) (MutableMapItem[K, R, V], bool) {
	item, found := m.m[key]
	if !found {
		item = m.r.Insert(rankIfCreated, key)
		m.m[key] = item
	}
	return m.mutableMapItem(item), found
}

func (m *Map[K, R, V]) Get(key K) MutableMapItem[K, R, V] {
	return m.mutableMapItem(m.m[key])
}

func (m *Map[K, R, V]) First() MutableMapItem[K, R, V] {
	return m.mutableMapItem(m.r.First())
}

func (m *Map[K, R, V]) Random(rnd *rand.Rand) MutableMapItem[K, R, V] {
	return m.mutableMapItem(m.r.Random(rnd))
}

func (m *Map[K, R, V]) RemoveOrdered() iter.Seq[*MapItem[K, R, V]] {
	return func(yield func(*MapItem[K, R, V]) bool) {
		for m.Len() > 0 {
			item := m.r.First()
			if !yield(mapItem(item)) {
				return
			}
			m.deleteItem(item)
		}
	}
}

func (m *Map[K, R, V]) Delete(key K) bool {
	if item := m.Get(key); item.Present() {
		item.Delete()
		return true
	}
	return false
}

func (m *Map[K, R, V]) DeleteFirst() {
	m.deleteItem(m.r.First())
}

func (m *Map[K, R, V]) deleteItem(item *rankedItem[K, R, V]) {
	delete(m.m, item.Auxiliary())
	m.r.Delete(item)
}
