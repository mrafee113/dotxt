package utils

func MkPtr[T any](val T) *T {
	return &val
}
