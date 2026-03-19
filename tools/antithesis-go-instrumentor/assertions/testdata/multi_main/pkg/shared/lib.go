package shared

import (
	"github.com/antithesishq/antithesis-sdk-go/assert"
)

func Init() {
	assert.Always(true, "shared assertion", nil)
}
