package sample

func Grid(n int) int {
	sum := 0
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if (i+j)%2 == 0 {
				sum += i * j
			}
		}
	}
	return sum
}
