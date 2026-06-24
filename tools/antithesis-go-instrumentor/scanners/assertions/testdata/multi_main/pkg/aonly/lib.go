package aonly

import (
	"github.com/antithesishq/antithesis-sdk-go/assert"
)

func Init() {
	assert.Sometimes(true, "aonly assertion", nil)
}
