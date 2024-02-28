package util

func IgnoreError(f func() error) {
	_ = f()
}
