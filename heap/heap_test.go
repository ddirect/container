package heap_test

import (
	"cmp"
	"encoding/json"
	"flag"
	"fmt"
	"iter"
	"math/rand/v2"
	"os"
	"slices"
	"testing"

	"github.com/ddirect/container/heap"
	"github.com/stretchr/testify/assert"
)

func Test_Basic(t *testing.T) {
	const n = 1000

	h := heap.New(func(a, b uint) bool {
		return a < b
	}, nil)

	var ref []uint
	for range n {
		v := rand.Uint()
		h.Push(v)
		ref = append(ref, v)
	}

	slices.Sort(ref)
	assert.Equal(t, ref, slices.Collect(h.PopAll()))
	assert.Equal(t, 0, h.Len())
}

type LogFunc func(t *testing.T, data []byte)

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

func makeCore(log LogFunc) func(t *testing.T, count, iterations int) {
	type node struct {
		val   uint
		index int
	}

	sortNodes := func(nodes []*node) {
		slices.SortFunc(nodes, func(a, b *node) int {
			return cmp.Compare(a.val, b.val)
		})
	}

	values := func(it iter.Seq[*node]) iter.Seq[uint] {
		return func(yield func(uint) bool) {
			for n := range it {
				if !yield(n.val) {
					return
				}
			}
		}
	}

	return func(t *testing.T, count, iterations int) {
		if count <= 0 || iterations <= 0 {
			return
		}

		// indexes of all items in heap
		var nodes []*node

		h := heap.New(func(a, b *node) bool {
			return a.val < b.val
		}, func(n *node, newIndex int) {
			n.index = newIndex
		})

		type stats struct {
			Count,
			Iterations,
			FinalLen, MaxLen, PushCount, FixCount, PopCount, RemoveCount int
		}

		s := &stats{
			Count:      count,
			Iterations: iterations,
		}

		push := func(count int) {
			for range count {
				n := &node{
					val: rand.Uint(),
				}
				h.Push(n)
				nodes = append(nodes, n)
				s.PushCount++
			}
			s.MaxLen = max(s.MaxLen, h.Len())
		}

		fix := func(count int) {
			if h.Len() < 2 {
				return
			}
			for range count {
				n := nodes[rand.IntN(len(nodes))]
				n.val = rand.Uint()
				h.Fix(n.index)
				s.FixCount++
			}
		}

		pop := func(t *testing.T, count int) {
			sortNodes(nodes)
			for range count {
				if h.Len() == 0 {
					return
				}
				assert.Equal(t, nodes[0], h.Pop())
				nodes = slices.Delete(nodes, 0, 1)
				s.PopCount++
			}
		}

		remove := func(t *testing.T, count int) {
			for range count {
				if h.Len() == 0 {
					return
				}
				i := rand.IntN(len(nodes))
				n := nodes[i]
				assert.Equal(t, n, h.Remove(n.index))
				nodes = slices.Delete(nodes, i, i+1)
				s.RemoveCount++
			}
		}

		for range iterations {
			switch rand.IntN(4) {
			case 0:
				push(rand.IntN(2 * count))
			case 1:
				fix(rand.IntN(count))
			case 2:
				pop(t, rand.IntN(count))
			case 3:
				remove(t, rand.IntN(count))
			}
		}

		s.FinalLen = h.Len()

		sStr, _ := json.Marshal(s)
		log(t, sStr)

		sortNodes(nodes)

		s1 := slices.Collect(values(slices.Values(nodes)))
		s2 := slices.Collect(values(h.PopAll()))

		assert.Equal(t, s1, s2)
		assert.Equal(t, 0, h.Len())
	}
}

func Fuzz_Multi(f *testing.F) {
	f.Add(10, 10000)
	f.Add(1000, 100)
	f.Fuzz(makeCore(makeLogFunc(logFile)))
}

var logFile string

func init() {
	flag.StringVar(&logFile, "logfile", "", "logfile to use")
}
