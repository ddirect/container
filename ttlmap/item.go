package ttlmap

import (
	"math"

	"github.com/ddirect/container/ranked"
)

type (
	Item[K comparable, V any]        = *ranked.MapItem[K, timestamp, V]
	mutableItem[K comparable, V any] = ranked.MutableMapItem[K, timestamp, V]
	MutableItem[K comparable, V any] struct {
		mutableItem[K, V]
	}
)

// func (it MutableItem[K, V]) Present() bool {
// 	return it.Rank() == timestamp(math.MaxInt64)
// }

// func (it *MutableItem[K, V]) makeNotPresent() {

// }
