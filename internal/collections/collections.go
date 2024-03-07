package collections

import (
	"sort"
)

func Convert[T any, P any](items []T, converterF func(T) P) []P {
	values := make([]P, 0, len(items))

	for _, item := range items {
		values = append(values, converterF(item))
	}

	return values
}

func Contains[T comparable](elems []T, elem T) bool {
	for _, e := range elems {
		if elem == e {
			return true
		}
	}
	return false
}

func ToMap[T any, K comparable](items []T, keyF func(T) K) map[K]T {
	m := make(map[K]T, len(items))

	for _, item := range items {
		k := keyF(item)
		m[k] = item
	}

	return m
}

func Keys[T any, K comparable](m map[K]T) []K {
	keys := make([]K, 0, len(m))

	for key := range m {
		keys = append(keys, key)
	}

	return keys
}

func Values[T any, K comparable](m map[K]T) []T {
	values := make([]T, 0, len(m))

	for _, v := range m {
		values = append(values, v)
	}

	return values
}

func SortBy[T any](items []T, lessF func(T, T) bool) []T {
	sort.Slice(items,
		func(i, j int) bool {
			return lessF(items[i], items[j])
		},
	)
	return items
}

func FilterBy[T any](items []T, filterF func(T) bool) []T {
	result := make([]T, 0)

	for _, item := range items {
		if filterF(item) {
			result = append(result, item)
		}
	}

	return result
}
