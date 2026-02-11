package collections

import (
	"sort"
)

const (
	UnlimitedBatchSize          = -1
	UnlimitedBatchPerGroupLimit = -1
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

func MapContains[K comparable, V any](data map[K]V, key K) bool {
	_, ok := data[key]
	return ok
}

func GroupByFunc[K comparable, V any](items []V, fn func(item V) K) map[K][]V {
	result := make(map[K][]V)
	for _, item := range items {
		k := fn(item)
		result[k] = append(result[k], item)
	}
	return result
}

func ToMap[T any, K comparable](items []T, keyF func(T) K) map[K]T {
	m := make(map[K]T, len(items))

	for _, item := range items {
		k := keyF(item)
		m[k] = item
	}

	return m
}

func Batch[K comparable, T any](items []T, batchSize, groupLimitPerBatch int, keyFn func(T) K) [][]T {
	type batchItem struct {
		elems []T
		keys  []K
	}

	batches := make(map[int]batchItem)

	for _, item := range items {
		groupKey := keyFn(item)

		for i := 0; true; i++ {
			if !MapContains(batches, i) {
				batches[i] = batchItem{elems: []T{}, keys: []K{}}
			}

			if len(batches[i].elems) >= batchSize && batchSize != UnlimitedBatchSize {
				continue
			}

			currentBatchItem := batches[i]
			if Contains(batches[i].keys, groupKey) {
				currentBatchItem.elems = append(currentBatchItem.elems, item)
				batches[i] = currentBatchItem

				break
			}

			// negative
			if len(currentBatchItem.keys) >= groupLimitPerBatch && groupLimitPerBatch != UnlimitedBatchPerGroupLimit {
				continue
			}

			currentBatchItem.elems = append(currentBatchItem.elems, item)
			currentBatchItem.keys = append(currentBatchItem.keys, groupKey)
			batches[i] = currentBatchItem

			break
		}
	}

	result := make([][]T, 0, len(batches))
	for _, bi := range batches {
		result = append(result, bi.elems)
	}

	return result
}

func ToIndexMap[T comparable](items []T) map[T]bool {
	m := make(map[T]bool, len(items))

	for _, item := range items {
		m[item] = true
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
