// Command switch_oneline exercises a switch (without a default) whose last
// case body ends on the same line as the switch's closing brace. The
// synthesized "default:" clause must not fuse with the preceding statement:
// `return "one" default:` does not parse, because automatic semicolon
// insertion only fires at line ends.
//
// NOTE: deliberately not gofmt-formatted -- the one-line switch is the point.
//
// Arg is a string whose length selects the case: program x
package main

import "os"

var sink string

func pick(n int) string {
	switch n { case 1: return "one" }
	return "many"
}

func main() {
	sink = pick(len(os.Args[1]))
}
