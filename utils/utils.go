package utils

func IsNumeric(value interface{}) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return true
	default:
		return false
	}
}

func Filter[T any](slice []T, qualifier func(T) bool) (result []T) {
	for _, s := range slice {
		if qualifier(s) {
			result = append(result, s)
		}
	}
	return
}

func TrimLeftChars(s string, n int) string {
	m := 0
	for i := range s {
		if m >= n {
			return s[i:]
		}
		m++
	}
	return s[:0]
}
