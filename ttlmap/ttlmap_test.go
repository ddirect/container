package ttlmap_test

import (
	"fmt"
	"iter"
	"log"
	"testing"
	"testing/synctest"
	"time"

	"github.com/ddirect/container/ttlmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	useSyncTest = true
	// NOTE: the max delay is 63 ticks
	ttlTicks     = 50
	accuracyTick = 5
	marginTicks  = 20
)

func testCore(t *testing.T, numKeys uint16, ops []byte) {
	if numKeys < 1 || len(ops) < 1 {
		return
	}

	const (
		tick     = time.Millisecond
		ttl      = ttlTicks * tick
		accuracy = accuracyTick * tick
	)
	margin := marginTicks * time.Millisecond
	if useSyncTest {
		margin = 0
	}

	core := func(t *testing.T) {
		log.SetFlags(log.LstdFlags | log.Lmicroseconds)
		log.Printf("ttl %v - accuracy %v", ttl, accuracy)

		ref := make(map[int]time.Time)
		m, expired := ttlmap.New[int, struct{}](ttl, accuracy)

		set := func(key int) {
			m.Set(key, struct{}{})
			ref[key] = time.Now()
		}

		get := func(key int) {
			item := m.Get(key)
			_, ok := ref[key]
			assert.Equal(t, item.Present(), ok)
			if ok {
				ref[key] = time.Now()
			} else {
				set(key)
			}
		}

		getOrCreate := func(key int) {
			_, ok := ref[key]
			_, found := m.GetOrCreate(key)
			assert.Equal(t, ok, found)
			ref[key] = time.Now()
		}

		remove := func(key int) {
			_, ok := ref[key]
			deleted := m.DeleteKey(key)
			assert.Equal(t, ok, deleted)
			if ok {
				delete(ref, key)
			} else {
				set(key)
			}
		}

		keyStore := uint16(0)
		timer := time.NewTimer(0) // start immediately
		nextOp := func(keyOffset int) (key int, method opMethod, ok bool) {
			if len(ops) >= 1 {
				keyStore++
				if keyStore >= numKeys {
					keyStore = 0
				}
				key = int(keyStore) + keyOffset<<16
				op := ops[0]
				ops = ops[1:]
				method = opMethod(op >> 6)
				sleepTime := tick * time.Duration(op&0x3F)
				log.Printf("op key %v method %v sleep %v", key, method, sleepTime)
				timer.Reset(sleepTime)
				ok = true
			}
			return
		}

		moreOps := true
		handleNextOp := func(keyOffset int) {
			if moreOps {
				if key, method, ok := nextOp(keyOffset); ok {
					switch method {
					case opSet:
						set(key)
					case opGet:
						get(key)
					case opGetOrCreate:
						getOrCreate(key)
					case opRemove:
						remove(key)
					}
				} else {
					moreOps = false
					log.Print("no more ops")
				}
			}
		}

		expiredKeyOffset := 1
		handleExpired := func(items iter.Seq[ttlmap.Item[int, struct{}]]) {
			for item := range items {
				elapsed := time.Since(ref[item.Key()])
				log.Printf("key %v expired after %v", item.Key(), elapsed)
				assert.GreaterOrEqual(t, elapsed, ttl-accuracy)
				assert.LessOrEqual(t, elapsed, ttl+accuracy+margin)
				delete(ref, item.Key())
				expiredKeyOffset++
			}
			// ensure only new keys are accessed
			handleNextOp(expiredKeyOffset)
		}

		// If the map timer and the test timer fire at the same time, only one of the two will be handled,
		// deadlocking the other, causing the test to fail. This can be seen with the following operations:
		//  set 0 - delay = ticks+accuracy.
		//  del 0 - 0 delay
		// In order to avoid this and other potential cases, the channels are flushed at the end of the test.
		flushChannels := func() {
			for {
				time.Sleep(100 * time.Millisecond)
				select {
				case <-timer.C:
					log.Print("test timer channel flushed")
				case <-expired:
					log.Print("expired items channel flushed")
				default:
					return
				}
			}
		}

		for moreOps || m.Len() > 0 {
			select {
			case <-timer.C:
				handleNextOp(0)
			case seq := <-expired:
				handleExpired(seq)
			}
		}
		flushChannels()
		assert.Empty(t, ref)
	}

	if useSyncTest {
		synctest.Test(t, core)
	} else {
		core(t)
	}
}

type opMethod int

const (
	opSet opMethod = iota
	opGet
	opGetOrCreate
	opRemove
)

func (m opMethod) String() string {
	switch m {
	case opSet:
		return "set"
	case opGet:
		return "get"
	case opGetOrCreate:
		return "getOrCreate"
	case opRemove:
		return "remove"
	default:
		panic(fmt.Errorf("invalid opMethod %d", m))
	}
}

type operations []byte

func (o *operations) reset() {
	*o = (*o)[:0]
}

func (o *operations) append(meth opMethod, delay byte) {
	*o = append(*o, byte(meth)<<6|delay)
}

func (o *operations) toBytes() []byte {
	return []byte(*o)
}

func Fuzz_ItemsExpireAtTheRightTime(f *testing.F) {
	var ops operations
	for delay := byte(95); delay <= 105; delay++ {
		ops.append(opSet, delay)
	}
	f.Add(uint16(1), ops.toBytes())

	ops.reset()
	for delay := range byte(110) {
		for meth := range 4 {
			ops.append(opMethod(meth), delay)
		}
	}
	f.Add(uint16(1), ops.toBytes())

	// test the stop timer code path
	ops.reset()
	ops.append(opSet, ttlTicks/2)
	ops.append(opRemove, 0)
	f.Add(uint16(1), ops.toBytes())

	f.Fuzz(testCore)
}

func Test_Touch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const ttl = time.Second
		m, expired := ttlmap.New[int, int](ttl, 0)

		var item [9]ttlmap.Item[int, int]
		var offset int

		assertValidData := func(i int) {
			assert.Equal(t, offset-i, *item[i].Value())
		}

		set := func(x ...int) {
			for _, i := range x {
				item[i] = m.Set(i, offset-i)
			}
		}

		touch := func(x ...int) {
			for _, i := range x {
				m.Touch(item[i])
			}
		}

		deleteItem := func(x ...int) {
			for _, i := range x {
				m.Delete(item[i])
			}
		}

		deleteKey := func(x ...int) {
			for _, i := range x {
				m.DeleteKey(i)
			}
		}

		requirePresent := func(x ...int) {
			for _, i := range x {
				require.True(t, item[i].Present())
				assertValidData(i)
			}
		}

		assertNotPresent := func(x ...int) {
			for _, i := range x {
				assert.False(t, item[i].Present())
				assertValidData(i)
			}
		}

		assertNeverSet := func(x ...int) {
			for _, i := range x {
				assert.False(t, item[i].Present())
				assert.Panics(t, func() { item[i].Key() })
				assert.Panics(t, func() { item[i].Value() })
				assert.Panics(t, func() { item[i].Rank() })
			}
		}

		waitAndAssertExpired := func(x ...int) {
			var exp []int
			for item := range <-expired {
				exp = append(exp, item.Key())
			}
			assert.ElementsMatch(t, x, exp)
		}

		/*
			0: do nothing
			1: set + expire
			2: set + touch + delete item
			3: set + touch + delete key
			4: set + touch + expire
			5: wait + set + touch + delete item
			6: wait + set + touch + delete key
			7: wait + set + expire
			8: wait + set + wait + touch + clear
		*/
		assertNeverSet(0, 1, 2, 3, 4, 5, 6, 7, 8)

		for offset = range 3 {
			t0 := time.Now()
			set(1, 2, 3, 4)
			requirePresent(1, 2, 3, 4)

			time.Sleep(ttl / 2)

			t1 := time.Now()
			requirePresent(1, 2, 3, 4)
			set(5, 6, 7, 8)
			touch(2, 3, 4)

			waitAndAssertExpired(1)
			assertNotPresent(1)
			assert.Equal(t, ttl, time.Since(t0))

			requirePresent(2, 3, 4, 5, 6, 7, 8)
			deleteItem(2, 5)
			deleteKey(3, 6)
			assertNotPresent(2, 3, 5, 6)
			requirePresent(4, 7, 8)

			time.Sleep(ttl / 2)
			requirePresent(4, 7, 8)
			touch(8)

			waitAndAssertExpired(4, 7)
			assertNotPresent(4, 7)
			assert.Equal(t, ttl, time.Since(t1))

			requirePresent(8)
			m.Clear()
			assertNotPresent(1, 2, 3, 4, 5, 6, 7, 8)
			assert.Zero(t, m.Len())
		}
	})
}
