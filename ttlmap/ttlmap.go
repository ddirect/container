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
	accuracyH    timestamp // accuracy/2
	queueCleanup func()
	timer        *time.Timer
}

// New creates a new ttlmap. ttl sets the minimum lifetime of each item. ttl must be at least 1ms; accuracy defines how much the item
// lifetime is allowed to be extended to avoid resetting the expiration timer. It must be less than ttl and can be 0.
// This version returns the map instance and a channel where item iterators are received. The iterators provide a notification on
// which items are expired. Iterating through the items is required in order for the items to be removed from the map.
func New[K comparable, V any](ttl, accuracy time.Duration) (*Map[K, V], <-chan iter.Seq[Item[K, V]]) {
	expired := make(chan iter.Seq[Item[K, V]])
	return NewAsync(ttl, accuracy, func(items iter.Seq[Item[K, V]]) {
		expired <- items
	}), expired
}

// NewAsync is like New, but instead of returning a channel, it gets a method which is called when items expire.
// Note that the returned iterator must not be used concurrently with other ttlmap methods, so proper syncrhonization
// must still be ensured externally.
func NewAsync[K comparable, V any](ttl, accuracy time.Duration, handleExpired func(iter.Seq[Item[K, V]])) *Map[K, V] {
	if ttl < time.Millisecond {
		panic(fmt.Errorf("ttlmap: invalid time-to-live: %v", ttl))
	}
	if accuracy >= ttl || accuracy < 0 {
		panic(fmt.Errorf("ttlmap: invalid accuracy %v for ttl %v", accuracy, ttl))
	}

	m := &Map[K, V]{
		m:         rankedmap.New[K, timestamp, V](),
		ttl:       fromDuration(ttl),
		accuracyH: fromDuration(accuracy / 2),
	}

	cleanup := func(yield func(Item[K, V]) bool) {
		now := getNow()
		for item := range m.m.RemoveOrdered() {
			// checkTimer expects that there are no items with expiration <= now
			if now.Before(item.Rank()) || !yield(item) {
				break
			}
		}
		if m.Len() > 0 {
			m.startTimer(now)
		} else {
			m.timer = nil // mark the timer as stopped
		}
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
	item, found := m.m.GetOrCreate(k, now+m.ttl+m.accuracyH)
	if found {
		m.refresh(item, now)
	}
	if m.timer == nil {
		m.startTimer(now)
	}
	return item, found
}

func (m *Map[K, V]) Delete(item Item[K, V]) {
	m.m.Delete(item)
}

func (m *Map[K, V]) DeleteKey(k K) bool {
	return m.m.DeleteKey(k)
}

func (m *Map[K, V]) Get(k K) Item[K, V] {
	now := getNow()
	item := m.m.Get(k)
	if item.Present() {
		m.refresh(item, now)
	}
	return item
}

func (m *Map[K, V]) GetNoTouch(k K) Item[K, V] {
	return m.m.Get(k)
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
	if item.Rank().Before(now + m.ttl) {
		m.m.SetRank(item, now+m.ttl+m.accuracyH)
	}
}

func (m *Map[K, V]) startTimer(now timestamp) {
	delay := m.m.First().Rank() - now + m.accuracyH
	m.timer = time.AfterFunc(toDuration(delay), m.queueCleanup)
}

/*
  Revised design:
  - rank when inserting: now+ttl+acc/2
  - rank range: from ttl to ttl+acc/2
  - timer delay: rank of first - now+acc/2
  - deletion condition when timer expires: rank <= now
  - resulting lifetime range: from ttl to ttl+acc

  Verification:
	time		action		life start	rank			note
	T0			insert I1	T0			T0+ttl+acc/2	timer starts with delay ttl+acc
	T0			insert I2	T0			T0+ttl+acc/2
	T0+acc/2	insert I3	T0+acc/2	T0+ttl+acc
	T0+acc/2	refresh I2	T0+acc/2	T0+ttl+acc/2	rank doesn't change
	T0+acc		refresh I3	T0+acc		T0+ttl+acc		rank doesn't change

  At T0+ttl+acc the timer expires; items with rank lower than this value are removed (in this case, all of them).
  Effective lifetimes:
	I1	ttl+acc
	I2	ttl+acc/2
	I3	ttl


  Original design:
  - rank when inserting: now+ttl
  - rank range: from ttl-acc to ttl
  - timer delay: rank of first + now+acc
  - deletion condition when timer expires: rank <= now
  - resulting lifetime range: from ttl-acc to ttl+acc

  Verification:
	time		action		life start	rank			note
	T0			insert I1	T0			T0+ttl			timer starts with delay ttl+acc
	T0			insert I2	T0			T0+ttl
	T0+acc		insert I3	T0+acc		T0+ttl+acc
	T0+acc		refresh I2	T0+acc		T0+ttl			rank doesn't change
	T0+2*acc	refresh I3	T0+2*acc	T0+ttl+acc		rank doesn't change

  At T0+ttl+acc the timer expires; items with rank lower than this value are removed (in this case, all of them).
  Effective lifetimes:
	I1	ttl+acc
	I2	ttl
	I3	ttl-acc
*/
