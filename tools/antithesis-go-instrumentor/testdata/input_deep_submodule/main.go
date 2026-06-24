package main

import (
	"fmt"

	"github.com/antithesishq/antithesis-sdk-go/assert"
)

func main() {
	fmt.Println("root")
	assert.Always(true, "root always", nil)
}
