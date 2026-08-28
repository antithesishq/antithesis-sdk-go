// Command branches exercises if / else-if chains and && / || short-circuit
// operators. It is side-effect free (no I/O beyond argv) and fully argv-driven,
// so every instrumented edge is reachable by some input:
//
//	program -5    // ||: left operand true, right short-circuited
//	program 200   // ||: right operand evaluated
//	program 4     // &&: both operands evaluated (even, nonzero)
//	program 0     // &&: right operand evaluated and false
//	program 3     // &&: left operand false, right short-circuited
package main

import (
	"os"
	"strconv"
)

// sink keeps the results observable-free work from being elided without any
// external side effect.
var sink string

func classify(n int) string {
	if n < 0 || n > 100 {
		return "out-of-range"
	}
	if n%2 == 0 && n != 0 {
		return "even"
	}
	return "odd-or-zero"
}

func main() {
	n, _ := strconv.Atoi(os.Args[1])
	sink = classify(n)
}
