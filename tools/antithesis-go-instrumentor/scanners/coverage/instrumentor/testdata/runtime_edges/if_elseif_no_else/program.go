// Command if_elseif_no_else exercises an `else if` chain with no trailing else.
// The instrumentor must place the synthesized "not taken" else inside the
// else-if wrapper block; getting it wrong produces a dangling else that won't
// compile. Args: one integer.  program 95 | program 85 | program 50
package main

import (
	"os"
	"strconv"
)

var sink string

func grade(n int) string {
	if n >= 90 {
		return "A"
	} else if n >= 80 {
		return "B"
	}
	return "C"
}

func main() {
	n, _ := strconv.Atoi(os.Args[1])
	sink = grade(n)
}
