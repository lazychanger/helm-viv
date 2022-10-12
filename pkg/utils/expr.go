package utils

func IF[T any](condition bool, trueVal T, falseVal T) T {

	if condition {
		return trueVal
	}
	return falseVal
}
