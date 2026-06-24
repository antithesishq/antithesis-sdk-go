package main

import (
	"fmt"

	__antithesis_instrumentation__ "antithesis.notifier/zad602425a68e"
	"github.com/antithesishq/antithesis-sdk-go/assert"
)

//line main.go:9
func foo(b bool) {
	__antithesis_instrumentation__.Notify(1)

//line main.go:10
	if b {
		__antithesis_instrumentation__.Notify(2)

//line main.go:11
		fmt.Println("b is true")

//line main.go:12
	} else {
		__antithesis_instrumentation__.Notify(3)

//line main.go:13
		fmt.Println("b is false")

//line main.go:14
		assert.Reachable("reached the else branch", nil)
	}
}

//line main.go:18
func main() {
	__antithesis_instrumentation__.Notify(4)

//line main.go:19
	fmt.Println("Hello, world!")

//line main.go:20
	assert.Always(true, "always in main", nil)

//line main.go:21
	foo(true)

//line main.go:22
	foo(false)
}
