package sample

func Classify(n int) string {
	if n < 0 {
		return "neg"
	} else if n == 0 {
		return "zero"
	} else {
		return "pos"
	}
}
