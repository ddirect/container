package ttlmap

import (
	"fmt"
	"iter"
	"time"

	"github.com/ddirect/container/ranked"
)

type timestamp time.Duration

func (t timestamp) Before(o timestamp) bool {
	return t < o
}

func fromDuration(t time.Duration) timestamp {
	return timestamp(t)
}

func toDuration(t timestamp) time.Duration {
	return time.Duration(t)
}

func fromTime(t time.Time) timestamp {
	return timestamp(t.UnixNano())
}

func getNow() timestamp {
	return fromTime(time.Now())
}

// ttlmap.Map is a key value store where unused items are automatically removed when they expire. It is not safe to call any method concurrently
// from different goroutines. This includes iterating on the expired items sequence.
type Map[K comparable, V any] struct {
	*ranked.Map[K, timestamp, V]
	ttl          timestamp
	accuracy     timestamp
	expired      chan iter.Seq[Item[K, V]]
	queueCleanup func()
	timer        *time.Timer
}

type (
	MutableItem[K comparable, V any] = ranked.MutableMapItem[K, timestamp, V]
	Item[K comparable, V any]        = *ranked.MapItem[K, timestamp, V]
)

// New creates a new ttlmap. ttl sets the coarse lifetime of each item. ttl must be at least 1ms; accuracy must be < ttl and can be 0.
// the lifetime is extended every time an item is read or written. accuracy is used for two purposes: firstly,
// the lifetime is not updated if the difference between the previous lifetime and the new lifetime is less than accuracy.
// Then, the timer used to check for expired elements is set with a duration which is above the set ttl by the accuracy time.
// In practice, ttl and accuracy define an expiration range: each item can be expected to expire between ttl-accuracy and ttl+accuracy
// plus any delay processing sequences Expired channel.
func New[K comparable, V any](ttl, accuracy time.Duration) *Map[K, V] {
	if ttl < time.Millisecond {
		panic(fmt.Errorf("ttlmap: invalid time-to-live: %v", ttl))
	}
	if accuracy >= ttl || accuracy < 0 {
		panic(fmt.Errorf("ttlmap: invalid accuracy %v for ttl %v", accuracy, ttl))
	}

	m := &Map[K, V]{
		Map:      ranked.NewMap[K, timestamp, V](),
		ttl:      fromDuration(ttl),
		accuracy: fromDuration(accuracy),
		expired:  make(chan iter.Seq[Item[K, V]]),
	}

	cleanup := func(yield func(Item[K, V]) bool) {
		now := getNow()
		for item := range m.RemoveOrdered() {
			// checkTimer expects that there are no items with expiration <= now
			if now.Before(item.Rank()) || !yield(item) {
				break
			}
		}
		m.timer = nil // mark the timer as stopped
		m.checkTimer(now)
	}

	// defining this here saves an allocation in the AfterFunc call
	m.queueCleanup = func() {
		m.expired <- cleanup
	}

	return m
}

func (m *Map[K, V]) Set(k K, v V) MutableItem[K, V] {
	item, _ := m.GetOrCreate(k)
	item.Value = v
	return item
}

func (m *Map[K, V]) GetOrCreate(k K) (MutableItem[K, V], bool) {
	now := getNow()
	item, found := m.Map.GetOrCreate(k, now+m.ttl)
	if found {
		m.refresh(item, now)
	}
	m.checkTimer(now)
	return item, found
}

func (m *Map[K, V]) Delete(k K) bool {
	if m.Map.Delete(k) {
		m.checkTimer(getNow())
		return true
	}
	return false
}

func (m *Map[K, V]) Get(k K) MutableItem[K, V] {
	now := getNow()
	item := m.Map.Get(k)
	if item.Present() {
		m.refresh(item, now)
	}
	return item
}

func (m *Map[K, V]) refresh(item MutableItem[K, V], now timestamp) {
	if item.Rank().Before(now + m.ttl - m.accuracy) {
		item.SetRank(now + m.ttl)
	}
}

// Expired returns a channel where iterators to expired items are returned.
// Receiving from the channel and using the iterator is required in order for the items to expire.
func (m *Map[K, V]) Expired() chan iter.Seq[Item[K, V]] {
	return m.expired
}

func (m *Map[K, V]) checkTimer(now timestamp) {
	if m.timer == nil && m.Len() > 0 {
		delay := m.Map.First().Rank() - now
		m.timer = time.AfterFunc(toDuration(delay+m.accuracy), m.queueCleanup)
	} else if m.timer != nil && m.Len() == 0 {
		if m.timer.Stop() {
			m.timer = nil
		}
	}
}
