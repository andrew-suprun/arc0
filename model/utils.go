package model

func Find[T any](slice []T, cond func(T) bool) (int, bool) {
	for i := range slice {
		if cond(slice[i]) {
			return i, true
		}
	}
	return 0, false
}
