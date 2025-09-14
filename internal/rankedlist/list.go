package rankedlist

import (
	"fmt"
	"iter"
	"math"
	"math/rand/v2"
	"slices"

	"github.com/ddirect/container"
)

type List[R container.Comparer[R], T any, A any] struct {
	s []*Item[R, T, A]
}

func (h *List[R, T, A]) Len() int {
	return len(h.s)
}

func (h *List[R, T, A]) ulen() uint {
	return uint(len(h.s))
}

func (h *List[R, T, A]) Clear() {
	clear(h.s)
}

func (h *List[R, T, A]) RemoveOrdered() iter.Seq[*Item[R, T, A]] {
	return func(yield func(*Item[R, T, A]) bool) {
		for h.Len() > 0 {
			item := h.First()
			if !yield(item) {
				return
			}
			h.Delete(item)
		}
	}
}

func (h *List[R, T, A]) Insert(rank R, aux A) *Item[R, T, A] {
	n := h.ulen()
	item := &Item[R, T, A]{
		rank:  rank,
		aux:   aux,
		index: n,
	}
	h.s = append(h.s, item)
	h.up(item)
	return item
}

func (h *List[R, T, A]) First() *Item[R, T, A] {
	return h.s[0]
}

func (h *List[R, T, A]) Random(rnd *rand.Rand) *Item[R, T, A] {
	return h.s[rnd.IntN(h.Len())]
}

func (h *List[R, T, A]) Values() iter.Seq[*Item[R, T, A]] {
	return slices.Values(h.s)
}

func (h *List[R, T, A]) DeleteFirst() {
	h.Delete(h.First())
}

func (h *List[R, T, A]) Delete(item *Item[R, T, A]) {
	n := h.ulen() - 1
	var last *Item[R, T, A]
	if n != item.index {
		if item.index > n {
			panic(fmt.Errorf("deleting item with index %d outside bounds", int(item.index)))
		}
		// take the last element and store it in place of the item to be deleted
		last = h.s[n]
		last.index = item.index
		h.s[last.index] = last
	}
	item.index = math.MaxUint // this is to cause a panic if this item is deleted again
	h.s[n] = nil
	h.s = h.s[:n]
	if last != nil && !h.down(last) {
		h.up(last)
	}
}

func (h *List[R, T, A]) SetRank(item *Item[R, T, A], rank R) {
	item.rank = rank
	if !h.down(item) {
		h.up(item)
	}
}

func (h *List[R, T, A]) parent(item *Item[R, T, A]) *Item[R, T, A] {
	if item.index == 0 {
		return nil
	}
	return h.s[(item.index-1)/2]
}

func (h *List[R, T, A]) up(item *Item[R, T, A]) {
	for {
		p := h.parent(item)
		if p == nil || !item.rank.Before(p.rank) {
			return
		}
		h.swap(p, item)
	}
}

func (h *List[R, T, A]) children(item *Item[R, T, A]) (c1 *Item[R, T, A], c2 *Item[R, T, A]) {
	i := 2*item.index + 1
	if i < h.ulen() {
		c1 = h.s[i]
		i++
		if i < h.ulen() {
			c2 = h.s[i]
		}
	}
	return
}

func (h *List[R, T, A]) down(item *Item[R, T, A]) bool {
	res := false
	for {
		c, c2 := h.children(item)
		if c == nil {
			return res
		}
		if c2 != nil && c2.rank.Before(c.rank) {
			c = c2
		}
		if item.rank.Before(c.rank) {
			return res
		}
		res = true
		h.swap(c, item)
	}
}

func (h *List[R, T, A]) swap(a, b *Item[R, T, A]) {
	a.index, b.index = b.index, a.index
	h.s[a.index] = a
	h.s[b.index] = b
}
