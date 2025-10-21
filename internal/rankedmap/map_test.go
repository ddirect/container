package rankedmap_test

import (
	"encoding/json"
	"maps"
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"github.com/ddirect/container/internal/rankedmap"
	"github.com/stretchr/testify/assert"
)

func Test_Basic(t *testing.T) {
	const n = 1000

	type (
		K uint32
		R = int32B
		V int64
	)

	m := rankedmap.New[K, R, V]()
	ref := make(map[K]refItem[K, R, V])

	for range n {
		k := rand.N[K](n)
		r := rand.N[R](n)
		v := V(rand.Uint64())

		_, found := ref[k]
		ref[k] = refItem[K, R, V]{k, r, v}

		item, loaded := m.GetOrCreate(k, r)
		assert.True(t, item.Present())
		assert.Equal(t, found, loaded)

		if loaded {
			m.SetRank(item, r)
		}
		*item.Value() = v

		assert.Equal(t, k, item.Key())
		assert.Equal(t, r, item.Rank())
	}

	// fully sort the reference by rank and then key
	s1 := slices.SortedFunc(maps.Values(ref), cmpRankThenKey)

	// same for All()
	s2 := slices.SortedFunc(toRefItems(t, m.All()), cmpRankThenKey)

	// use native sorting for ranks and then only sort by key if the rank is the same
	s3 := slices.Collect(toRefItems(t, m.RemoveOrdered()))
	slices.SortStableFunc(s3, cmpOnlyKeyIfRankSame)

	assert.Equal(t, s1, s2)
	assert.Equal(t, s1, s3)
	assert.Equal(t, 0, m.Len())
}

func Test_Delete(t *testing.T) {
	type (
		K uint32
		R = time.Time
		V int64
	)

	m := rankedmap.New[K, R, V]()

	time1 := time.Unix(1, 0)
	time2 := time.Unix(2, 0)
	time3 := time.Unix(3, 0)

	item1, found := m.GetOrCreate(0, time1)
	assert.True(t, item1.Present())
	assert.False(t, found)
	item1_rank := item1.Rank()

	item2, found := m.GetOrCreate(0, time2)
	assert.True(t, item1 == item2)
	assert.True(t, found)
	assert.True(t, item2.Rank().Equal(item1_rank))

	item3, found := m.GetOrCreate(1, time3)
	assert.True(t, item3.Present())
	assert.False(t, found)

	assert.True(t, m.DeleteKey(0))
	assert.False(t, item1.Present())
	assert.False(t, m.DeleteKey(0))
	assert.True(t, m.DeleteKey(1))
	assert.False(t, item3.Present())
	assert.False(t, m.DeleteKey(2))
}

func Test_Clear(t *testing.T) {
	type (
		K int
		R = int32B
		V int64
	)

	m := rankedmap.New[K, R, V]()

	const count = 1000
	for k := range K(count) {
		m.Set(k, R(k), V(k))
	}

	for k := range K(count) {
		assert.True(t, m.Exists(k))
		assert.True(t, m.Get(k).Present())
		assert.Equal(t, R(k), m.Get(k).Rank())
		assert.Equal(t, V(k), *m.Get(k).Value())
	}

	assert.Equal(t, count, m.Len())
	m.Clear()
	assert.Zero(t, m.Len())

	for k := range K(count) {
		assert.False(t, m.Exists(k))
		assert.False(t, m.Get(k).Present())
	}
}

func Test_RemoveOrdered(t *testing.T) {
	type (
		K = int
		R = int32B
		V int64
	)

	m := rankedmap.New[K, R, V]()

	m.Set(3, 3, 3)
	m.Set(2, 2, 2)
	m.Set(1, 1, 1)

	for item := range m.RemoveOrdered() {
		assert.True(t, item.Present())
		assert.Equal(t, 3, m.Len())
		assert.Equal(t, 1, item.Key())
		break
	}

	first := true
	for item := range m.RemoveOrdered() {
		assert.True(t, item.Present())
		m.Delete(item)
		assert.False(t, item.Present())
		if first {
			assert.Equal(t, 2, m.Len())
			assert.Equal(t, 1, item.Key())
			first = false
		} else {
			assert.Equal(t, 1, m.Len())
			assert.Equal(t, 2, item.Key())
			break
		}
	}

	for item := range m.RemoveOrdered() {
		assert.True(t, item.Present())
		assert.Equal(t, 1, m.Len())
		assert.Equal(t, 3, item.Key())
		break
	}

	assert.Equal(t, 1, m.Len())
}

func makeCore(log LogFunc) func(t *testing.T, seed uint64, variance int) {
	type (
		K int32
		R = int32B
		V uint32
	)

	type stats struct {
		Seed uint64
		Variance,
		MaxKey, MaxRank, Iterations,
		FinalLen, MaxLen,
		Set, GetOrCreateNew, GetOrCreateExisting, DeleteRandom, DeleteFirst, SetRank int
	}

	var (
		t                           *testing.T
		rnd                         *rand.Rand
		maxKey, maxRank, iterations int
		s                           stats
	)
	ref := make(map[K]*refItem[K, R, V])
	m := rankedmap.New[K, R, V]()

	create := func() bool {
		k := K(rnd.IntN(maxKey))
		r := R(rnd.IntN(maxRank))
		v := V(rnd.Uint64())

		m.Set(k, r, v)
		ref[k] = &refItem[K, R, V]{k, r, v}
		s.Set++

		s.MaxLen = max(s.MaxLen, m.Len())
		return true
	}

	getOrCreate := func() bool {
		k := K(rnd.IntN(maxKey))
		r := R(rnd.IntN(maxRank))
		v := V(rnd.Uint64())

		item, loaded := m.GetOrCreate(k, r)
		ri := ref[k]
		assert.Equal(t, ri != nil, loaded)
		if ri == nil {
			ri = new(refItem[K, R, V])
			ref[k] = ri
			ri.rank = r
			s.GetOrCreateNew++
		} else {
			s.GetOrCreateExisting++
		}
		*item.Value() = v
		ri.key = k
		ri.value = v

		s.MaxLen = max(s.MaxLen, m.Len())
		return true
	}

	deleteRandom := func() bool {
		if m.Len() == 0 {
			return false
		}
		item := m.Random(rnd)
		m.Delete(item)
		delete(ref, item.Key())
		s.DeleteRandom++
		return true
	}

	deleteFirst := func() bool {
		if m.Len() == 0 {
			return false
		}
		item := m.First()
		m.Delete(item)
		delete(ref, item.Key())
		s.DeleteFirst++
		return true
	}

	setRank := func() bool {
		if m.Len() == 0 {
			return false
		}
		r := R(rnd.IntN(maxRank))
		item := m.Random(rnd)
		m.SetRank(item, r)
		ref[item.Key()].rank = r
		s.SetRank++
		return true
	}

	runMulti := func(core func() bool) {
		for range rnd.IntN(10) + 1 {
			if iterations <= 0 || !core() {
				return
			}
			iterations--
		}
	}

	return func(t_ *testing.T, seed uint64, variance int) {
		if variance < 1 {
			return
		}

		clear(ref)

		t = t_
		rnd = rand.New(rand.NewPCG(seed, 0))
		maxKey = rnd.IntN(variance) + 1
		maxRank = rnd.IntN(variance) + 1
		iterations = rnd.IntN(variance) + 1
		s = stats{
			Seed:       seed,
			Variance:   variance,
			MaxKey:     maxKey,
			MaxRank:    maxRank,
			Iterations: iterations,
		}

		for iterations > 0 {
			if m.Len() == 0 {
				runMulti(getOrCreate)
			} else {
				switch rnd.IntN(8) {
				case 0:
					runMulti(deleteRandom)
				case 1:
					runMulti(deleteFirst)
				case 2:
					runMulti(setRank)
				case 3, 4:
					runMulti(create)
				default:
					runMulti(getOrCreate)
				}
			}
		}

		s.FinalLen = m.Len()

		sStr, _ := json.Marshal(s)
		log(t, sStr)

		// fully sort the reference by rank and then key
		s1 := slices.SortedFunc(dereference(maps.Values(ref)), cmpRankThenKey)

		// use native sorting for ranks and then only sort by key if the rank is the same
		s2 := slices.Collect(toRefItems(t, m.RemoveOrdered()))
		slices.SortStableFunc(s2, cmpOnlyKeyIfRankSame)

		assert.Equal(t, s1, s2)
		assert.Equal(t, 0, m.Len())
	}
}

func Fuzz_Multi(f *testing.F) {
	f.Add(uint64(1), 10)
	f.Add(uint64(2), 1000)
	f.Fuzz(makeCore(makeLogFunc(logFile)))
}
