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
	useSyncTest   = true
	methBits      = 3
	delayBits     = 8 - methBits
	maxDelay      = 1<<delayBits - 1
	ttlTicks      = 25
	accuracyTicks = 5
	marginTicks   = 10
)

func testCore(t *testing.T, numKeys uint16, ops []byte) {
	if numKeys < 1 || len(ops) < 1 {
		return
	}

	const (
		tick     = time.Millisecond
		ttl      = ttlTicks * tick
		accuracy = accuracyTicks * tick
	)
	margin := marginTicks * tick
	if useSyncTest {
		margin = 0
	}

	core := func(t *testing.T) {
		log.SetFlags(log.LstdFlags | log.Lmicroseconds)
		log.Printf("ttl %v - accuracy %v", ttl, accuracy)

		ref := make(map[int]time.Time)
		m, expired := ttlmap.New[int, struct{}](ttl, accuracy)

		storedItem := m.NullItem()
		storedKey := -1

		set := func(key int, count int) {
			for range count {
				storedItem = m.Set(key, struct{}{})
				storedKey = key
				ref[key] = time.Now()
				key++ // NOTE: key may extend above numKeys here
			}
		}

		get := func(key int) {
			_, ok := ref[key]
			item := m.Get(key)
			assert.Equal(t, ok, item.Present())
			if ok {
				ref[key] = time.Now()
			} else {
				set(key, 1)
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
				if key == storedKey {
					storedKey = -1
				}
			} else {
				set(key, 1)
			}
		}

		touchItem := func(key int) {
			_, ok := ref[storedKey]
			present := storedItem.Present()
			assert.Equal(t, ok, present)
			if present {
				m.Touch(storedItem)
				ref[storedKey] = time.Now()
			} else {
				set(key, 1)
			}
		}

		removeItem := func(key int) {
			_, ok := ref[storedKey]
			present := storedItem.Present()
			assert.Equal(t, ok, present)
			if present {
				m.Delete(storedItem)
				delete(ref, storedKey)
				storedKey = -1
			} else {
				set(key, 1)
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
				method = opMethod(op >> delayBits)
				sleepTime := tick * time.Duration(op&0x1F)
				log.Printf("op key %v storedkey %v method %v sleep %v", key, storedKey, method, sleepTime)
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
					case opSet1:
						set(key, 1)
					case opSet2:
						set(key, 2)
					case opSet3:
						set(key, 3)
					case opGet:
						get(key)
					case opGetOrCreate:
						getOrCreate(key)
					case opRemove:
						remove(key)
					case opTouchItem:
						touchItem(key)
					case opRemoveItem:
						removeItem(key)
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
				assert.GreaterOrEqual(t, elapsed, ttl)
				assert.LessOrEqual(t, elapsed, ttl+accuracy+margin)
				delete(ref, item.Key())
				if item.Key() == storedKey {
					storedKey = -1
				}
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
	opSet1 opMethod = iota
	opSet2
	opSet3
	opGet
	opGetOrCreate
	opRemove
	opTouchItem
	opRemoveItem
)

func (m opMethod) String() string {
	switch m {
	case opSet1:
		return "set1"
	case opSet2:
		return "set2"
	case opSet3:
		return "set3"
	case opGet:
		return "get"
	case opGetOrCreate:
		return "getOrCreate"
	case opRemove:
		return "remove"
	case opTouchItem:
		return "touchItem"
	case opRemoveItem:
		return "removeItem"
	default:
		panic(fmt.Errorf("invalid opMethod %d", m))
	}
}

type operations []byte

func (o *operations) reset() {
	*o = (*o)[:0]
}

func (o *operations) append(meth opMethod, delay byte) {
	if delay > maxDelay {
		panic(fmt.Errorf("requested delay %d more than max %d", delay, maxDelay))
	}
	*o = append(*o, byte(meth)<<delayBits|delay)
}

func (o *operations) toBytes() []byte {
	return []byte(*o)
}

func Fuzz_ItemsExpireAtTheRightTime(f *testing.F) {
	var ops operations
	for delay := byte(20); delay <= 30; delay++ {
		ops.append(opSet1, delay)
	}
	f.Add(uint16(1), ops.toBytes())

	ops.reset()
	for delay := range byte(1 << delayBits) {
		for meth := range 1 << methBits {
			ops.append(opMethod(meth), delay)
		}
	}
	f.Add(uint16(1), ops.toBytes())

	// test the stop timer code path
	ops.reset()
	ops.append(opSet1, ttlTicks/2)
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

func Test_GetNoTouch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const ttl = time.Second
		m, expired := ttlmap.New[int, int](ttl, 0)
		assertExpired := func(expected int) {
			count := 0
			select {
			case seq := <-expired:
				for range seq {
					count++
				}
			default:
			}
			assert.Equal(t, expected, count)
		}

		m.Set(0, 0)

		time.Sleep(ttl * 2 / 3)
		assertExpired(0)
		assert.True(t, m.Get(0).Present())

		time.Sleep(ttl * 2 / 3)
		assertExpired(0)
		assert.True(t, m.GetNoTouch(0).Present())

		time.Sleep(ttl * 2 / 3)
		assertExpired(1)
		assert.False(t, m.Get(0).Present())
	})
}
