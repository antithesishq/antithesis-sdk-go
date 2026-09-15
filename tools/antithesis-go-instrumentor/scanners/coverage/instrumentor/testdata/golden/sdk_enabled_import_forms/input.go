package sample

import (
	// A dot import cannot qualify a selector, but it must not stop the search
	// for a usable alias further down the list.
	. "github.com/antithesishq/antithesis-sdk-go/assert"
	// A raw-string import path spells the same package.
	raw `github.com/antithesishq/antithesis-sdk-go/assert`
	// The same package may be bound to more than one name.
	second "github.com/antithesishq/antithesis-sdk-go/assert"
)

// Recognized through a raw-string import path.
func Raw(n int) int {
	if raw.Enabled {
		n++
	}
	return n
}

// Recognized through a second alias for the same package.
func Second(n int) int {
	if second.Enabled {
		n++
	}
	return n
}

// The dot-imported name is not attributable in an AST-only pass, so this keeps
// its "not taken" edge.
func Dotted(n int) int {
	if Enabled {
		n++
	}
	return n
}
