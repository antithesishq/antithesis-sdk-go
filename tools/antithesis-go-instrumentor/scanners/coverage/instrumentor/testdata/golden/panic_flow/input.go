package sample

// A call to panic ends a basic block, so the statements before and after it
// land in separate edges.
func Do() {
	println("before")
	panic("boom")
}
