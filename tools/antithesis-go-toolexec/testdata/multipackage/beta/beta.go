package beta

import "multipkg/shared"

func Total(n int) int {
	total := 0
	for i := 0; i < n; i++ {
		if i%3 == 0 {
			total += shared.Scale(i)
		}
	}
	return total
}
