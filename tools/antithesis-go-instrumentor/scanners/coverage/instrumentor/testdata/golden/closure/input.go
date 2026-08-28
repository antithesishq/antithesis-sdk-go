package sample

func Make() func() int {
	n := 0
	return func() int {
		n++
		return n
	}
}
