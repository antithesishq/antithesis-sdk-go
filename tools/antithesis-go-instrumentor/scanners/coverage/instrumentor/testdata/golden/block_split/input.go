package sample

// The loop body splits into multiple edges: the branch statement (break)
// ends a basic block, so the statements before and after it are counted
// separately.
func Scan(xs []int) int {
	count := 0
	for _, x := range xs {
		count++
		if x < 0 {
			break
		}
		count++
	}
	return count
}
