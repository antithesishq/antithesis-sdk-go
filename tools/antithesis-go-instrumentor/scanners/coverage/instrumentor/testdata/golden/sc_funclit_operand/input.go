// The right operand of && is a func literal with control flow of its own.
// The short-circuit rewrite must instrument inside the literal's body (the
// AST-rewriting backend does), not copy it verbatim.
package sample

func hasBig(a bool, n int) bool {
	return a && func() bool {
		if n > 1 {
			return true
		}
		return false
	}()
}
