// Command sc_operand_subtrees puts instrumentable code *inside* short-circuit
// operands, where the source-editing backend's rewriteBool only recurses
// through ParenExpr and &&/|| BinaryExpr nodes:
//
//   - funcLitOperand: a func literal (with an if/else of its own) on the right
//     of &&. The AST-rewriting backend walks into the literal's body and
//     instruments it; a backend that copies the operand verbatim records no
//     edges inside it.
//   - nestedUnderNot: a || chain nested under ! (a unary operator, so it is
//     not reachable through the &&/|| chain itself).
//
// Both backends must emit identical symbol tables and fire identical edges.
//
// Args: program <a t|f> <nstr> <a2 t|f> <b t|f> <c t|f>
// where len(nstr) is the n passed to funcLitOperand.
package main

import "os"

var sink bool

func funcLitOperand(a bool, n int) bool {
	return a && func() bool {
		if n > 1 {
			return true
		}
		return false
	}()
}

func nestedUnderNot(a, b, c bool) bool {
	return a && !(b || c)
}

func main() {
	sink = funcLitOperand(os.Args[1] == "t", len(os.Args[2]))
	sink = nestedUnderNot(os.Args[3] == "t", os.Args[4] == "t", os.Args[5] == "t")
}
