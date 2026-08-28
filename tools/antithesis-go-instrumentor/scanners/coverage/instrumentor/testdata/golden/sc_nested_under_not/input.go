// A || chain nested under a unary ! is not reachable through the outer
// &&/|| chain that rewriteBool recurses over; the operand's subtree must
// still be instrumented rather than copied verbatim.
package sample

func neither(a, b, c bool) bool {
	return a && !(b || c)
}
