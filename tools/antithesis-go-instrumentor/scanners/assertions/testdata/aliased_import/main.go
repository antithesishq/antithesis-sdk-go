package main

import (
	a "github.com/antithesishq/antithesis-sdk-go/assert"
)

func main() {
	a.Always(true, "aliased always", nil)
	a.Unreachable("aliased unreachable", nil)
}
