package heap

import (
	"iter"
)

type Heap[T any] struct {
	s        []T
	lessFunc func(a, b T) bool
	newIndex func(t T, i int)
}

func New[T any](less func(a, b T) bool, newIndex func(t T, i int)) *Heap[T] {
	return &Heap[T]{
		lessFunc: less,
		newIndex: newIndex,
	}
}

func (h *Heap[T]) Len() int {
	return len(h.s)
}

func (h *Heap[T]) Get(i int) T {
	return h.s[i]
}

func (h *Heap[T]) PopAll() iter.Seq[T] {
	return func(yield func(T) bool) {
		for h.Len() > 0 {
			if !yield(h.Pop()) {
				return
			}
		}
	}
}

func (h *Heap[T]) Push(x T) {
	h.s = append(h.s, x)
	n := h.Len() - 1
	if !h.up(n) && h.newIndex != nil {
		h.newIndex(x, n)
	}
}

func (h *Heap[T]) Pop() T {
	n := h.Len() - 1
	h.swap(0, n)
	h.down(0, n)
	return h.pop(n)
}

func (h *Heap[T]) Remove(i int) T {
	n := h.Len() - 1
	if n != i {
		h.swap(i, n)
		if !h.down(i, n) {
			h.up(i)
		}
	}
	return h.pop(n)
}

func (h *Heap[T]) Fix(i int) {
	if !h.down(i, h.Len()) {
		h.up(i)
	}
}

func (h *Heap[T]) up(j0 int) bool {
	j := j0
	for {
		i := (j - 1) / 2 // parent
		if i == j || !h.less(j, i) {
			break
		}
		h.swap(i, j)
		j = i
	}
	return j < j0
}

func (h *Heap[T]) down(i0, n int) bool {
	i := i0
	for {
		j1 := 2*i + 1
		if j1 >= n || j1 < 0 { // j1 < 0 after int overflow
			break
		}
		j := j1 // left child
		if j2 := j1 + 1; j2 < n && h.less(j2, j1) {
			j = j2 // = 2*i + 2  // right child
		}
		if !h.less(j, i) {
			break
		}
		h.swap(i, j)
		i = j
	}
	return i > i0
}

func (h *Heap[T]) swap(i, j int) {
	a, b := h.s[i], h.s[j]
	h.s[j], h.s[i] = a, b
	if h.newIndex != nil {
		h.newIndex(a, j)
		h.newIndex(b, i)
	}
}

func (h *Heap[T]) pop(n int) T {
	e := h.s[n]
	h.s = h.s[:n]
	return e
}

func (h *Heap[T]) less(i, j int) bool {
	return h.lessFunc(h.s[i], h.s[j])
}
