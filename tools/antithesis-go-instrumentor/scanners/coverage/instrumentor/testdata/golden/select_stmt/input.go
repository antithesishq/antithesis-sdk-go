package sample

func Recv(a, b chan int) int {
	select {
	case v := <-a:
		return v
	case v := <-b:
		return v
	}
}
