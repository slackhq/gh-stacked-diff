// Utility functions (inspired by Kotlin) for slices.
package util

// Returns a slice containing the results of applying the given transform function to each element
// in the original array.
func MapSlice[V, R any](slice []V, f func(V) R) []R {
	mapped := make([]R, len(slice))
	for i, s := range slice {
		mapped[i] = f(s)
	}
	return mapped
}

// Returns a slice containing only elements matching the given predicate.
func FilterSlice[V any](slice []V, f func(V) bool) []V {
	filtered := make([]V, 0, len(slice))
	for _, s := range slice {
		if f(s) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}
