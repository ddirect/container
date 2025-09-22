package ttlmap

import (
	"fmt"
	"iter"
	"time"

	"github.com/ddirect/container/internal/rankedmap"
)

// ttlmap.Map is a key value store where unused items are automatically removed when they expire. It is not safe to call any method concurrently
// from different goroutines. This includes iterating on the expired items sequence.
type Map[K comparable, V any] struct {
	m            *rankedmap.Map[K, timestamp, V]
	ttl          timestamp
	accuracy     timestamp
	queueCleanup func()
	timer        *time.Timer
}

// New creates a new ttlmap. ttl sets the coarse lifetime of each item. ttl must be at least 1ms; accuracy must be < ttl and can be 0.
// the lifetime is extended every time an item is read or written. accuracy is used for two purposes: firstly,
// the lifetime is not updated if the difference between the previous lifetime and the new lifetime is less than accuracy.
// Then, the timer used to check for expired elements is set with a duration which is above the set ttl by the accuracy time.
// In practice, ttl and accuracy define an expiration range: each item can be expected to expire between ttl-accuracy and ttl+accuracy
// plus any delay processing the sequence of expired items.
// This version returns the map instance and a channel where to receive the iterators which must be processed in order for the items to be
// removed from the map.
func New[K comparable, V any](ttl, accuracy time.Duration) (*Map[K, V], <-chan iter.Seq[Item[K, V]]) {
	expired := make(chan iter.Seq[Item[K, V]])
	return NewAsync(ttl, accuracy, func(items iter.Seq[Item[K, V]]) {
		expired <- items
	}), expired
}

// NewAsync is like New, but instead of returning a channel, it gets a handling method which is called when expired items
// have to be processed. Note that the iterator must not be called concurrently with other ttlmap methods, so proper syncrhonization
// must still be ensured externally.
func NewAsync[K comparable, V any](ttl, accuracy time.Duration, handleExpired func(iter.Seq[Item[K, V]])) *Map[K, V] {
	if ttl < time.Millisecond {
		panic(fmt.Errorf("ttlmap: invalid time-to-live: %v", ttl))
	}
	if accuracy >= ttl || accuracy < 0 {
		panic(fmt.Errorf("ttlmap: invalid accuracy %v for ttl %v", accuracy, ttl))
	}

	m := &Map[K, V]{
		m:        rankedmap.New[K, timestamp, V](),
		ttl:      fromDuration(ttl),
		accuracy: fromDuration(accuracy),
	}

	cleanup := func(yield func(Item[K, V]) bool) {
		now := getNow()
		for item := range m.m.RemoveOrdered() {
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
		handleExpired(cleanup)
	}

	return m
}

func (m *Map[K, V]) NullItem() Item[K, V] {
	return Item[K, V]{}
}

func (m *Map[K, V]) Len() int {
	return m.m.Len()
}

func (m *Map[K, V]) Set(k K, v V) Item[K, V] {
	item, _ := m.GetOrCreate(k)
	*item.Value() = v
	return item
}

func (m *Map[K, V]) GetOrCreate(k K) (Item[K, V], bool) {
	now := getNow()
	item, found := m.m.GetOrCreate(k, now+m.ttl)
	if found {
		m.refresh(item, now)
	}
	m.checkTimer(now)
	return item, found
}

func (m *Map[K, V]) Delete(item Item[K, V]) {
	m.m.Delete(item)
}

func (m *Map[K, V]) DeleteKey(k K) bool {
	if m.m.DeleteKey(k) {
		m.checkTimer(getNow())
		return true
	}
	return false
}

func (m *Map[K, V]) Get(k K) Item[K, V] {
	now := getNow()
	item := m.m.Get(k)
	if item.Present() {
		m.refresh(item, now)
	}
	return item
}

func (m *Map[K, V]) Exists(k K) bool {
	return m.m.Exists(k)
}

func (m *Map[K, V]) All() iter.Seq[Item[K, V]] {
	return m.m.All()
}

func (m *Map[K, V]) Clear() {
	m.m.Clear()
}

func (m *Map[K, V]) Touch(item Item[K, V]) {
	m.refresh(item, getNow())
}

func (m *Map[K, V]) refresh(item Item[K, V], now timestamp) {
	if item.Rank().Before(now + m.ttl - m.accuracy) {
		m.m.SetRank(item, now+m.ttl)
	}
}

func (m *Map[K, V]) checkTimer(now timestamp) {
	if m.timer == nil && m.Len() > 0 {
		delay := m.m.First().Rank() - now
		m.timer = time.AfterFunc(toDuration(delay+m.accuracy), m.queueCleanup)
	} else if m.timer != nil && m.Len() == 0 { // TODO: see if stopping the timer is really necessary: it looks like it's not
		if m.timer.Stop() {
			m.timer = nil
		}
	}
}
