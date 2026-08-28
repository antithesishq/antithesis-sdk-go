package sample

// No function bodies means no edges: Instrument returns "" and the caller
// copies the file verbatim. The expected.golden is therefore empty.
const (
	A = 1
	B = 2
)

type Point struct {
	X, Y int
}
