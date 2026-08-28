package alpha

import "multipkg/shared"

func Describe(n int) string {
	if n < 0 {
		return "negative:" + shared.Classify(-n)
	}
	return "positive:" + shared.Classify(n)
}
