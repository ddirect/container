package ttlmap

import (
	"time"

	"github.com/ddirect/container/ranked"
)

type timestamp int64

func (t timestamp) Before(o timestamp) bool {
	return t < o
}

func fromDuration(t time.Duration) timestamp {
	return timestamp(t / time.Millisecond)
}

func fromTime(t time.Time) timestamp {
	return timestamp(t.UnixMilli())
}

type Map[K comparable, V any] struct {
	*ranked.Map[K, timestamp, V]
	ttl      timestamp
	accuracy timestamp
	deleted  func(Item[K, V])
}

type (
	MutableItem[K comparable, V any] = ranked.MutableMapItem[K, timestamp, V]
	Item[K comparable, V any]        = *ranked.MapItem[K, timestamp, V]
)

func New[K comparable, V any](ttl, accuracy time.Duration, deleted func(Item[K, V])) *Map[K, V] {
	return &Map[K, V]{
		Map:      ranked.NewMap[K, timestamp, V](),
		ttl:      fromDuration(ttl),
		accuracy: fromDuration(accuracy),
		deleted:  deleted,
	}
}

func (m *Map[K, V]) GetOrCreate(k K) (MutableItem[K, V], bool) {
	now := m.cleanup()
	item, found := m.Map.GetOrCreate(k, now)
	if found {
		m.refresh(item, now)
	}
	return item, found
}

func (m *Map[K, V]) Get(k K) MutableItem[K, V] {
	now := m.cleanup()
	item := m.Map.Get(k)
	if item.Present() {
		m.refresh(item, now)
	}
	return item
}

func (m *Map[K, V]) refresh(item MutableItem[K, V], now timestamp) {
	if item.Rank().Before(now - m.accuracy) {
		item.SetRank(now)
	}
}

func (m *Map[K, V]) cleanup() timestamp {
	now := fromTime(time.Now())
	limit := now - m.ttl
	for item := range m.RemoveOrdered() {
		if !item.Rank().Before(limit) {
			break
		}
		if m.deleted != nil {
			m.deleted(item)
		}
	}
	return now
}
