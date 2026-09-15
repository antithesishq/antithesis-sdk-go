package sample

import "github.com/antithesishq/antithesis-sdk-go/assert"

type toggles struct {
	Enabled bool
}

// The parameter shadows the imported package name, so this is an ordinary
// runtime condition and its "not taken" edge must still be recorded. Matching
// the import alias as text alone would wrongly delete it.
func Shadowed(assert toggles, n int) int {
	if assert.Enabled {
		n++
	}
	return n
}

// At file scope the same spelling really is the SDK constant.
func NotShadowed(n int) int {
	if assert.Enabled {
		n++
	}
	return n
}
