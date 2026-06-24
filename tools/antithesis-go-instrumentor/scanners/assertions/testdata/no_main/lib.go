package lib

import (
	"github.com/antithesishq/antithesis-sdk-go/assert"
)

func Check() {
	assert.Always(true, "library assertion", nil)
}
