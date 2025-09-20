package rankedmap

import (
	"iter"
	"math/rand/v2"

	"github.com/ddirect/container"
	"github.com/ddirect/container/internal/rankedlist"
)

type Map[K comparable, R container.Comparer[R], V any] struct {
	r *rankedList[K, R, V]
	m map[K]*rankedItem[K, R, V]
}

func New[K comparable, R container.Comparer[R], V any]() *Map[K, R, V] {
	return &Map[K, R, V]{
		r: rankedlist.New[R, kv[K, V]](),
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

func (m *Map[K, R, V]) Set(key K, rank R, value V) MapItem[K, R, V] {
	item, found := m.m[key]
	if found {
		m.r.SetRank(item, rank)
	} else {
		item = m.r.Insert(rank)
		item.Value().key = key
		m.m[key] = item
	}
	item.Value().value = value
	return mapItem(item)
}

func (m *Map[K, R, V]) GetOrCreate(key K, rankIfCreated R) (MapItem[K, R, V], bool) {
	item, found := m.m[key]
	if !found {
		item = m.r.Insert(rankIfCreated)
		item.Value().key = key
		m.m[key] = item
	}
	return mapItem(item), found
}

func (m *Map[K, R, V]) Get(key K) MapItem[K, R, V] {
	return mapItem(m.m[key])
}

func (m *Map[K, R, V]) Exists(key K) bool {
	_, ok := m.m[key]
	return ok
}

func (m *Map[K, R, V]) First() MapItem[K, R, V] {
	return mapItem(m.r.First())
}

func (m *Map[K, R, V]) Random(rnd *rand.Rand) MapItem[K, R, V] {
	return mapItem(m.r.Random(rnd))
}

func (m *Map[K, R, V]) All() iter.Seq[MapItem[K, R, V]] {
	return func(yield func(MapItem[K, R, V]) bool) {
		for it := range m.r.Values() {
			if !yield(mapItem(it)) {
				return
			}
		}
	}
}

func (m *Map[K, R, V]) RemoveOrdered() iter.Seq[MapItem[K, R, V]] {
	return func(yield func(MapItem[K, R, V]) bool) {
		for m.Len() > 0 {
			item := m.r.First()
			if !yield(mapItem(item)) {
				return
			}
			m.deleteItem(item)
		}
	}
}

func (m *Map[K, R, V]) SetRank(it MapItem[K, R, V], rank R) {
	m.r.SetRank(it.rankedItem, rank)
}

func (m *Map[K, R, V]) Delete(it MapItem[K, R, V]) {
	m.deleteItem(it.rankedItem)
}

func (m *Map[K, R, V]) DeleteKey(key K) bool {
	if item := m.Get(key); item.Present() {
		m.Delete(item)
		return true
	}
	return false
}

func (m *Map[K, R, V]) DeleteFirst() {
	m.deleteItem(m.r.First())
}

func (m *Map[K, R, V]) deleteItem(item *rankedItem[K, R, V]) {
	delete(m.m, item.Value().key)
	m.r.Delete(item)
}
