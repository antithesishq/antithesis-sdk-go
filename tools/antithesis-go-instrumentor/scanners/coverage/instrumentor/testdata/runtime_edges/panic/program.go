// Command panic exercises a panic-terminated basic block (panic ends a block,
// so statements before and after it are separate edges) and the recovery path.
// Side-effect free and argv-driven:
//
//	program boom   // takes the panic; the post-panic edge must NOT fire
//	program ok     // no panic; the post-panic edge fires
package main

import "os"

var sink int

func run(boom bool) {
	sink = 1
	if boom {
		panic("boom")
	}
	sink = 2
}

func main() {
	defer func() { _ = recover() }()
	run(os.Args[1] == "boom")
}
