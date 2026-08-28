package shared

var Eager = Classify(7)

func Classify(n int) string {
	if n > 5 {
		return "high"
	}
	if n > 0 {
		return "low"
	}
	return "none"
}

func Scale(n int) int {
	if n%2 == 0 {
		return n * 2
	}
	return n * 3
}
