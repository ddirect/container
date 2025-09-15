package rankedlist_test

import (
	"cmp"
	"iter"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/ddirect/container/internal/rankedlist"
	"github.com/stretchr/testify/assert"
)

type int32B int32

func (a int32B) Before(b int32B) bool {
	return a < b
}

func Test_Basic(t *testing.T) {
	const (
		n    = 1000
		maxR = 1000
		maxT = 1000
		maxA = 1000
	)

	type (
		R = int32B
		T int
		A int64
	)

	var h rankedlist.List[R, T, A]

	type refItem struct {
		rank  R
		value T
		aux   A
	}

	toRefItems := func(it iter.Seq[*rankedlist.Item[R, T, A]]) iter.Seq[refItem] {
		return func(yield func(refItem) bool) {
			for i := range it {
				if !yield(refItem{i.Rank(), i.Value, i.Auxiliary()}) {
					return
				}
			}
		}
	}

	var ref []refItem
	for range n {
		r := rand.N[R](maxR)
		v := rand.N[T](maxT)
		a := rand.N[A](maxA)
		item := h.Insert(r, a)
		item.Value = v
		assert.Equal(t, r, item.Rank())
		assert.Equal(t, a, item.Auxiliary())
		ref = append(ref, refItem{r, v, a})
	}

	slices.SortFunc(ref, func(a, b refItem) int {
		if a.rank == b.rank {
			if a.value == b.value {
				return cmp.Compare(a.aux, b.aux)
			}
			return cmp.Compare(a.value, b.value)
		}
		return cmp.Compare(a.rank, b.rank)
	})

	s := slices.Collect(toRefItems(h.RemoveOrdered()))
	slices.SortStableFunc(s, func(a, b refItem) int {
		if a.rank == b.rank {
			if a.value == b.value {
				return cmp.Compare(a.aux, b.aux)
			}
			return cmp.Compare(a.value, b.value)
		}
		return 0
	})

	assert.Equal(t, ref, s)
	assert.Equal(t, 0, h.Len())
}

func Test_CannotDeleteTwice(t *testing.T) {
	var h rankedlist.List[int32B, int32, int32]
	item := h.Insert(0, 0)
	h.Insert(1, 1)
	h.Delete(item)
	assert.Panics(t, func() { h.Delete(item) })
}

func Test_Values(t *testing.T) {
	const count = 1000
	var h rankedlist.List[int32B, struct{}, int]
	for i := range count {
		h.Insert(int32B(rand.Int32()), i+1)
	}

	ref := make([]int, count)
	for val := range h.Values() {
		i := val.Auxiliary() - 1
		assert.Zero(t, ref[i])
		ref[i] = i + 1
	}

	for i, v := range ref {
		assert.Equal(t, i+1, v)
	}
}

func Test_Present(t *testing.T) {
	var h rankedlist.List[int32B, struct{}, int]
	item := h.Insert(1, 0)
	assert.True(t, item.Present())
	h.Delete(item)
	assert.False(t, item.Present())
}
