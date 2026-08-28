package sample

// Sign has no else branch; the instrumentor synthesizes one so the
// not-taken path gets coverage.
func Sign(n int) int {
	if n > 0 {
		return 1
	}
	return 0
}
