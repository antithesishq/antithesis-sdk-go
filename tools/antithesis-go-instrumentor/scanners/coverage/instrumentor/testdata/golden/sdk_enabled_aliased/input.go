package sample

import ant "github.com/antithesishq/antithesis-sdk-go/assert"

type toggles struct {
	Enabled bool
}

var feature toggles

// An explicit import alias is honored.
func Aliased(n int) int {
	if ant.Enabled {
		n++
	}
	return n
}

// A selector on something else that merely spells the same way is not the SDK
// constant, and keeps its "not taken" edge.
func Other(n int) int {
	if feature.Enabled {
		n++
	}
	return n
}

// The undotted name is not bound to the assert package here at all.
func Unrelated(n int) int {
	if n > 0 {
		ant.Reachable("positive", nil)
	}
	return n
}
