// Command sc_or_chain exercises a flat || chain: a || b || c. The right operand
// of each || is instrumented, and short-circuit evaluation means a given
// operand's edge fires only if every operand to its left was false.
// Args are three booleans as "t"/"f":  program f t f
package main

import "os"

var sink bool

func chain(a, b, c bool) bool {
	return a || b || c
}

func main() {
	sink = chain(os.Args[1] == "t", os.Args[2] == "t", os.Args[3] == "t")
}
