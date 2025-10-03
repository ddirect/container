package ttlmap

import (
	"encoding/binary"
	"iter"
	"maps"
	"math/rand/v2"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// returns the counter of expired items by the actual expiration time, in ticks
func insertCore(t *testing.T, ttlTicks, accuracyTicks int, delays []byte) (count map[int]int) {
	count = make(map[int]int)
	synctest.Test(t, func(t *testing.T) {
		const tick = time.Millisecond
		ttl := time.Duration(ttlTicks) * tick
		accuracy := time.Duration(accuracyTicks) * tick

		var mutex sync.Mutex
		ref := make(map[int]time.Time)

		totalExpired := 0
		m := NewAsync(ttl, accuracy, func(expired iter.Seq[Item[int, int]]) {
			mutex.Lock()
			defer mutex.Unlock()
			for item := range expired {
				elapsed := time.Since(ref[item.Key()])
				require.Zero(t, elapsed%tick)
				count[int(elapsed/tick)]++
				totalExpired++
			}
			assert.NotZero(t, count)
		})

		totalInserted := 0
		var last Item[int, int]
		getOrCreate := func(key int) (found bool) {
			mutex.Lock()
			defer mutex.Unlock()

			last, found = m.GetOrCreate(key)
			if !found {
				totalInserted++
			}
			ref[key] = time.Now()
			return
		}

		touchLast := func() bool {
			mutex.Lock()
			defer mutex.Unlock()

			if !last.Present() {
				return false
			}
			m.Touch(last)
			ref[last.Key()] = time.Now()
			return true
		}

		for key, delay := range delays {
			time.Sleep(time.Duration(delay) * tick)
			touchLast() // even with the touch operation the item may not change rank
			assert.False(t, getOrCreate(key))
		}

		time.Sleep(ttl + accuracy + tick)
		require.Equal(t, totalInserted, totalExpired)
	})
	return
}

func Fuzz_TimerFiresNoMoreThanNeeded(f *testing.F) {
	var buf []byte
	for i := 255; i >= 0; i-- {
		buf = append(buf, byte(i))
	}
	f.Add(uint8(127), buf)
	f.Fuzz(func(t *testing.T, ttlTicksB uint8, delays []byte) {
		if len(delays) > 0 {
			ttlTicks := int(ttlTicksB) + 1
			accuracyTicks := ttlTicks / 10

			countMap := insertCore(t, ttlTicks, accuracyTicks, delays)

			exp := slices.Sorted(maps.Keys(countMap))
			assert.GreaterOrEqual(t, exp[0], ttlTicks)
			assert.LessOrEqual(t, exp[len(exp)-1], ttlTicks+accuracyTicks)
		}
	})
}

func Test_TimerFiresNoMoreThanNeeded(t *testing.T) {
	var seed [32]byte
	binary.LittleEndian.PutUint64(seed[:], uint64(time.Now().UnixNano()))
	rnd := rand.NewChaCha8(seed)

	// a lot of iterations are needed to catch a couple of items in the first map bucket (expiring at ttl)
	delays := make([]byte, 300000)

	core := func(ttlTicks, accuracyTicks int) {
		rnd.Read(delays)
		countMap := insertCore(t, ttlTicks, accuracyTicks, delays)

		counts := make([]int, accuracyTicks+1)
		for exp, count := range countMap {
			assert.NotZero(t, count)
			counts[exp-ttlTicks] = count
		}

		for i, count := range counts {
			t.Logf("%10d%10d", i, count)
			assert.NotZerof(t, count, "zero items expired %d ticks after ttl", i)
		}
	}

	core(1000, 30)
}

func Test_ItemIsNotAlwaysRefreshed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const ttl = time.Second
		const accuracy = ttl / 10
		m, _ := New[int, int](ttl, accuracy)
		item := m.Set(0, 0)
		rank := item.Rank()

		time.Sleep(accuracy / 4)
		m.Touch(item) // rank not updated
		assert.Equal(t, rank, item.Rank())

		time.Sleep(accuracy / 4)
		m.Touch(item) // rank not updated
		assert.Equal(t, rank, item.Rank())

		const step = 1
		time.Sleep(step)
		m.Touch(item) // rank updated
		assert.Equal(t, rank+m.accuracyH+step, item.Rank())
	})
}
