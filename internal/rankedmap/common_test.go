package rankedmap_test

import (
	"cmp"
	"flag"
	"fmt"
	"iter"
	"os"
	"testing"

	"github.com/ddirect/container"
	"github.com/ddirect/container/internal/rankedmap"
)

type LogFunc func(t *testing.T, data []byte)

var logFile string

func init() {
	flag.StringVar(&logFile, "logfile", "", "logfile to use")
}

func makeLogFunc(logFile string) LogFunc {
	if logFile == "" {
		return func(t *testing.T, data []byte) {
			t.Logf("%s\n", data)
		}
	}

	logout, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic(fmt.Errorf("open: %w", err))
	}

	return func(t *testing.T, data []byte) {
		if _, err := logout.Write(append(data, '\n')); err != nil {
			panic(fmt.Errorf("write: %w", err))
		}
	}
}

type comparerWithCompare[T any] interface {
	container.Comparer[T]
	Compare(T) int
}

type int32B int32

func (a int32B) Before(b int32B) bool {
	return a < b
}

func (a int32B) Compare(b int32B) int {
	return cmp.Compare(a, b)
}

type refItem[K cmp.Ordered, R comparerWithCompare[R], V any] struct {
	key   K
	rank  R
	value V
}

func cmpRankThenKey[K cmp.Ordered, R comparerWithCompare[R], V any](a, b refItem[K, R, V]) int {
	rcmp := a.rank.Compare(b.rank)
	if rcmp == 0 {
		return cmp.Compare(a.key, b.key)
	}
	return rcmp
}

func cmpOnlyKeyIfRankSame[K cmp.Ordered, R comparerWithCompare[R], V any](a, b refItem[K, R, V]) int {
	if a.rank.Compare(b.rank) == 0 {
		return cmp.Compare(a.key, b.key)
	}
	return 0
}

func toRefItems[K cmp.Ordered, R comparerWithCompare[R], V any](it iter.Seq[rankedmap.MapItem[K, R, V]]) iter.Seq[refItem[K, R, V]] {
	return func(yield func(refItem[K, R, V]) bool) {
		for i := range it {
			if !yield(refItem[K, R, V]{i.Key(), i.Rank(), *i.Value()}) {
				return
			}
		}
	}
}

func dereference[T any](it iter.Seq[*T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		for i := range it {
			if !yield(*i) {
				return
			}
		}
	}
}
