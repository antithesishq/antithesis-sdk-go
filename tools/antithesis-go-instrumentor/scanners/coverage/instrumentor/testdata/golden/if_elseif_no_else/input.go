package sample

// Grade has an `else if` with no trailing else, so the instrumentor synthesizes
// the "not taken" else for the else-if's inner if. That synthesized else must
// land inside the wrapper block, not after it (else: dangling else).
func Grade(n int) string {
	if n >= 90 {
		return "A"
	} else if n >= 80 {
		return "B"
	}
	return "C"
}
