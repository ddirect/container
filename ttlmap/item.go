package ttlmap

import "github.com/ddirect/container/internal/rankedmap"

type Item[K comparable, V any] = rankedmap.MapItem[K, timestamp, V]
