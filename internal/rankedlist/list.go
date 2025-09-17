package rankedlist

import (
	"fmt"
	"iter"
	"math/rand/v2"
	"slices"

	"github.com/ddirect/container"
)

type List[R container.Comparer[R], T any] struct {
	s []*Item[R, T]
}

func (h *List[R, T]) Len() int {
	return len(h.s)
}

func (h *List[R, T]) ulen() uint {
	return uint(len(h.s))
}

func (h *List[R, T]) Clear() {
	clear(h.s)
	h.s = h.s[:0]
}

func (h *List[R, T]) RemoveOrdered() iter.Seq[*Item[R, T]] {
	return func(yield func(*Item[R, T]) bool) {
		for h.Len() > 0 {
			item := h.First()
			if !yield(item) {
				return
			}
			h.Delete(item)
		}
	}
}

func (h *List[R, T]) Insert(rank R) *Item[R, T] {
	n := h.ulen()
	item := &Item[R, T]{
		rank:    rank,
		indexP1: n + 1,
	}
	h.s = append(h.s, item)
	h.up(item)
	return item
}

func (h *List[R, T]) First() *Item[R, T] {
	return h.s[0]
}

func (h *List[R, T]) Random(rnd *rand.Rand) *Item[R, T] {
	return h.s[rnd.IntN(h.Len())]
}

func (h *List[R, T]) Values() iter.Seq[*Item[R, T]] {
	return slices.Values(h.s)
}

func (h *List[R, T]) DeleteFirst() {
	h.Delete(h.First())
}

func (h *List[R, T]) Delete(item *Item[R, T]) {
	n := h.ulen() - 1
	var last *Item[R, T]
	i := item.indexP1 - 1
	if n != i {
		if i > n {
			panic(fmt.Errorf("deleting item with index %d outside bounds", int(i)))
		}
		// take the last element and store it in place of the item to be deleted
		last = h.s[n]
		last.indexP1 = i + 1
		h.s[i] = last
	}
	item.setNotPresent()
	h.s[n] = nil
	h.s = h.s[:n]
	if last != nil && !h.down(last) {
		h.up(last)
	}
}

func (h *List[R, T]) SetRank(item *Item[R, T], rank R) {
	item.rank = rank
	if !h.down(item) {
		h.up(item)
	}
}

func (h *List[R, T]) parent(item *Item[R, T]) *Item[R, T] {
	i := item.indexP1 - 1
	if i == 0 {
		return nil
	}
	return h.s[(i-1)/2]
}

func (h *List[R, T]) up(item *Item[R, T]) {
	for {
		p := h.parent(item)
		if p == nil || !item.rank.Before(p.rank) {
			return
		}
		h.swap(p, item)
	}
}

func (h *List[R, T]) children(item *Item[R, T]) (c1 *Item[R, T], c2 *Item[R, T]) {
	i := 2*item.indexP1 - 1 // i := 2*realIndex + 1 = 2*(item.indexP1-1) + 1 = 2*item.indexP1 - 1
	if i < h.ulen() {
		c1 = h.s[i]
		i++
		if i < h.ulen() {
			c2 = h.s[i]
		}
	}
	return
}

func (h *List[R, T]) down(item *Item[R, T]) bool {
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

func (h *List[R, T]) swap(a, b *Item[R, T]) {
	a.indexP1, b.indexP1 = b.indexP1, a.indexP1
	h.s[a.indexP1-1] = a
	h.s[b.indexP1-1] = b
}
