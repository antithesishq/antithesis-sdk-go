package sample

import "github.com/antithesishq/antithesis-sdk-go/assert"

// The positive, no-else guard. assert.Enabled is a constant, so a synthesized
// else could never be taken; instrumenting it would leave a row in the symbol
// table that can never be notified.
func Guarded(n int) {
	if assert.Enabled {
		assert.Always(n >= 0, "n is not negative", nil)
	}
}

// An ordinary condition in the same file still gets its synthesized else.
func Ordinary(n int) string {
	if n > 0 {
		return "pos"
	}
	return "other"
}

// An explicit else is left alone: there is nothing to synthesize.
func ExplicitElse(n int) string {
	if assert.Enabled {
		assert.Reachable("checked", nil)
		return "checked"
	} else {
		return "unchecked"
	}
}

// The negated form is deliberately not special-cased. Here the body is the dead
// branch and the synthesized else is live, so suppressing the else would delete
// a real edge.
func Negated(n int) string {
	if !assert.Enabled {
		return "disabled"
	}
	assert.Sometimes(n == 0, "n is zero", nil)
	return "enabled"
}

// As the else-if of another statement, where the enclosing wrapper's closing
// brace is inserted at the same offset the synthesized else would have used.
func ElseIf(n int) string {
	if n < 0 {
		return "neg"
	} else if assert.Enabled {
		assert.Always(n >= 0, "n is not negative in else-if", nil)
		return "nonneg"
	}
	return "quiet"
}
