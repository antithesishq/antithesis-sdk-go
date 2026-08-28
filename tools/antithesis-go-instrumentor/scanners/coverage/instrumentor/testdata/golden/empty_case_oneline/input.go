// An empty case clause written on the same line as the following clause:
// `case 0: case 1: ...`. The instrumentor inserts a Notify after every
// `case:`; for the empty clause it must keep a separator, or the result fuses
// into `Notify(e) case 1:`, which does not parse (automatic semicolon
// insertion only fires at line ends). Deliberately not gofmt-formatted.
package sample

func pick(n int) int {
	switch n { case 0: case 1: return 1 }
	return 0
}
