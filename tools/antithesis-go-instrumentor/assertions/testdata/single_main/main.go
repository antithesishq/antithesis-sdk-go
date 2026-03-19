package main

import (
	"github.com/antithesishq/antithesis-sdk-go/assert"
)

func main() {
	assert.Always(true, "always true", nil)
	assert.Sometimes(true, "sometimes true", nil)
	assert.Reachable("reached main", nil)
}
