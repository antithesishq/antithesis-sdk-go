// Command empty_case_oneline exercises an empty case clause written on the
// same line as the following clause: `case 0: case 1: return 1`. The
// instrumentor inserts a Notify after each `case:`; an empty clause abutting
// the next one must keep a separator (`Notify(e); case 1:`) or the
// instrumented program does not parse -- automatic semicolon insertion only
// fires at line ends.
//
// Deliberately not gofmt-formatted -- the one-line empty case is the point.
//
// Arg is a string whose length selects the case: program x
package main

import "os"

var sink int

func pick(n int) int {
	switch n { case 0: case 1: return 1 }
	return 0
}

func main() {
	sink = pick(len(os.Args[1]))
}
