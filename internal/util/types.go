package util

func Pointer[T any](o T) *T {
	return &o
}
