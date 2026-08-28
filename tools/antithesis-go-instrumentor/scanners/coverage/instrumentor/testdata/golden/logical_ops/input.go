package sample

// Both/Either exercise the && and || rewrite: the right operand is wrapped
// in an instrumented closure compared against the intrinsic true.
func Both(a, b bool) bool {
	return a && b
}

func Either(a, b bool) bool {
	return a || b
}
