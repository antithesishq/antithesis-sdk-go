// name's switch has no default and its case body ends on the same line as
// the switch's closing brace. The synthesized "default:" clause needs a
// leading separator or it fuses with the preceding statement (`return "one"
// default:` does not parse -- auto-semicolon insertion only fires at line
// ends). Deliberately not gofmt-formatted; the one-line switch is the point.
package sample

func name(n int) string {
	switch n { case 1: return "one" }
	return "many"
}
