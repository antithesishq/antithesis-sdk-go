package main

import (
	"fmt"

	"example.com/multimodule-test/sub"
	"github.com/antithesishq/antithesis-sdk-go/assert"
)

func main() {
	fmt.Println("hello from root")
	assert.Always(true, "root always", nil)
	sub.DoIt()
}
