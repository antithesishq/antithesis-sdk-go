//go:build !antithesis_ignore
// +build !antithesis_ignore

package sample

// Classify has a body to instrument. The point of this case is the build
// constraints above: the instrumentor must keep both the //go:build and the
// legacy // +build lines verbatim, and must place its own leading //line
// directive ABOVE them. A //line is itself a line comment, which the build-tag
// scanner skips over, so the constraints still gate compilation. The
// antithesis_ignore tag is never set, so the constraints are always satisfied
// and the file always compiles.
func Classify(n int) string {
	if n < 0 {
		return "negative"
	}
	return "nonneg"
}
