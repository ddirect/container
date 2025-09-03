package set

import "iter"

type Set[T comparable] map[T]struct{}

func New[T comparable]() Set[T] {
	return Set[T](make(map[T]struct{}))
}

func (s Set[T]) Insert(t T) {
	s[t] = struct{}{}
}

func (s Set[T]) Delete(t T) {
	delete(s, t)
}

func (s Set[T]) Exists(t T) bool {
	_, ok := s[t]
	return ok
}

func (s Set[T]) Len() int {
	return len(s)
}

func (s Set[T]) Values() iter.Seq[T] {
	return func(yield func(T) bool) {
		for t := range s {
			if !yield(t) {
				return
			}
		}
	}
}
