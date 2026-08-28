// Command sc_paren_mid exercises a parenthesized group nested inside an &&
// chain: a && (b || c) && d. This is the recursive-rewrite case -- the group
// (b || c) is itself a short-circuit expression, and c is instrumented inside
// it, so the instrumentor must descend through the ParenExpr.
// Args are four booleans as "t"/"f":  program t f t t
package main

import "os"

var sink bool

func chain(a, b, c, d bool) bool {
	return a && (b || c) && d
}

func main() {
	sink = chain(os.Args[1] == "t", os.Args[2] == "t", os.Args[3] == "t", os.Args[4] == "t")
}
