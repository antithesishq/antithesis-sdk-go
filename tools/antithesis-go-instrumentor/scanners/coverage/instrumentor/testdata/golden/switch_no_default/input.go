package sample

// Name's switch has no default; the instrumentor appends an empty default
// case to capture the fall-through path.
func Name(n int) string {
	switch n {
	case 1:
		return "one"
	case 2:
		return "two"
	}
	return "many"
}
