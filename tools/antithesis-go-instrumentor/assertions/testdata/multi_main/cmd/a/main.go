package main

import (
	"github.com/antithesishq/antithesis-sdk-go/assert"
	_ "github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/assertions/testdata/multi_main/pkg/aonly"
	_ "github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/assertions/testdata/multi_main/pkg/shared"
)

func main() {
	assert.Always(true, "a main assertion", nil)
}
