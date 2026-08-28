//go:build !antithesis_ignore
// +build !antithesis_ignore

// Command build_constraint proves that an instrumented file whose build
// constraints are always satisfied still compiles and runs after the
// instrumentor prepends its leading //line directive ABOVE the //go:build and
// // +build lines. (The golden case build_constraint checks the static
// placement; this checks that the compiler still accepts and includes the
// file.) The antithesis_ignore tag is never set, so the constraints hold on
// every platform. Args: one integer.  program 5 | program -5
package main

import (
	"os"
	"strconv"
)

var sink string

func classify(n int) string {
	if n < 0 {
		return "negative"
	}
	return "nonneg"
}

func main() {
	n, _ := strconv.Atoi(os.Args[1])
	sink = classify(n)
}
