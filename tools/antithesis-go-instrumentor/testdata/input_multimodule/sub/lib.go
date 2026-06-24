package sub

import (
	"fmt"

	"github.com/antithesishq/antithesis-sdk-go/assert"
)

func DoIt() {
	fmt.Println("hello from sub")
	assert.Reachable("sub reached", nil)
}
