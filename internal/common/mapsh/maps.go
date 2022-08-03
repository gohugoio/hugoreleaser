package mapsh

import (
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
)

// KeysSorted returns the keys of the map sorted.
func KeysSorted[M ~map[K]V, K constraints.Ordered, V any](m M) []K {
	keys := KeysComparable(m)
	slices.Sort(keys)
	return keys
}

// KeysComparable returns the keys of the map m.
// The keys will be in an indeterminate order but K needs to be ordered.
func KeysComparable[M ~map[K]V, K constraints.Ordered, V any](m M) []K {
	r := make([]K, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	return r
}
