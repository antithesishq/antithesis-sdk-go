package sample

var ready bool

// init is deliberately NOT instrumented -- it runs regardless, so counting
// it is just noise. Ready should still be instrumented.
func init() {
	ready = true
}

func Ready() bool {
	return ready
}
