package inner

import (
	"github.com/antithesishq/antithesis-sdk-go/assert"
)

func DoIt() {
	assert.Reachable("inner reached", nil)
}
