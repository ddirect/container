package fifo

type Fifo[T any] struct {
	s []T
}

func (f *Fifo[T]) Enqueue(t T) {
	f.s = append(f.s, t)
}

func (f *Fifo[T]) Dequeue() (t T, ok bool) {
	if len(f.s) > 0 {
		t = f.s[0]
		f.s = f.s[1:]
		ok = true
	}
	return
}

func (f *Fifo[T]) Len() int {
	return len(f.s)
}
