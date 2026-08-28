package sample

import _ "embed"

// before sits ahead of the //go:embed var, mirroring the old ticket's repro
// (a function precedes the directive). The instrumentor rewrites this
// function; we then check that the embed below still populates the var.
func before() {
	println("before")
}

//go:embed hello.txt
var hello string

// Sum ranges over the embedded string (as the ticket did with `range hello`)
// and returns the byte sum, exercising instrumentation right after the embed.
func Sum() int {
	total := 0
	for _, b := range hello {
		total += int(b)
	}
	return total
}

// Hello returns the embedded contents so a test can confirm the embed
// actually happened -- not just that the file compiled.
func Hello() string {
	return hello
}
