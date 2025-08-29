package container

type Comparer[T any] interface {
	Before(T) bool
}
