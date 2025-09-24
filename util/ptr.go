package util

// Ptr returns a pointer to the given value
func Ptr[T any](v T) *T {
	return &v
}

// PtrIf returns a pointer to the value if the condition is met, otherwise nil
func PtrIf[T any](v T, condition bool) *T {
	if condition {
		return &v
	}
	return nil
}

// PtrIfNotZero returns a pointer to the value if it's not the zero value, otherwise nil
func PtrIfNotZero[T comparable](v T) *T {
	var zero T
	if v != zero {
		return &v
	}
	return nil
}

// PtrIfNotEmpty returns a pointer to the string if it's not empty, otherwise nil
func PtrIfNotEmpty(s string) *string {
	if s != "" {
		return &s
	}
	return nil
}

// ValueOr returns the value pointed to by ptr, or defaultValue if ptr is nil
func ValueOr[T any](ptr *T, defaultValue T) T {
	if ptr != nil {
		return *ptr
	}
	return defaultValue
}

// Value returns the value pointed to by ptr, or the zero value if ptr is nil
func Value[T any](ptr *T) T {
	var zero T
	if ptr != nil {
		return *ptr
	}
	return zero
}
