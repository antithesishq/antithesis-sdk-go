// Command sc_paren_both exercises parenthesized groups on both sides of an &&:
// (a || b) && (c || d). Both groups are short-circuit expressions the
// instrumentor must descend into, and the right group is itself the right
// operand of the outer && (so it is wrapped and also recursed into).
// Args are four booleans as "t"/"f":  program t f t f
package main

import "os"

var sink bool

func chain(a, b, c, d bool) bool {
	return (a || b) && (c || d)
}

func main() {
	sink = chain(os.Args[1] == "t", os.Args[2] == "t", os.Args[3] == "t", os.Args[4] == "t")
}
