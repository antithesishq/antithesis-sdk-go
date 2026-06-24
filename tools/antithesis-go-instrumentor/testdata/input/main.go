package main

import (
	"fmt"

	"github.com/antithesishq/antithesis-sdk-go/assert"
)

func foo(b bool) {
	if b {
		fmt.Println("b is true")
	} else {
		fmt.Println("b is false")
		assert.Reachable("reached the else branch", nil)
	}
}

func main() {
	fmt.Println("Hello, world!")
	assert.Always(true, "always in main", nil)
	foo(true)
	foo(false)
}
