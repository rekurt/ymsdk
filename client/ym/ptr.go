package ym

// Ptr returns a pointer to v. It is a convenience helper for populating
// optional pointer fields in request structs.
func Ptr[T any](v T) *T {
	return &v
}
