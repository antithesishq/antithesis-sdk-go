//line main.go:1:1
package main; import __antithesis_instrumentation__ "antithesis.notifier/zad602425a68e"

import (
	"fmt"

	"github.com/antithesishq/antithesis-sdk-go/assert"
)

func foo(b bool) {__antithesis_instrumentation__.Notify(1)
	if b {__antithesis_instrumentation__.Notify(2)
		fmt.Println("b is true")
	} else {__antithesis_instrumentation__.Notify(3)
		fmt.Println("b is false")
		assert.Reachable("reached the else branch", nil)
	}
}

func main() {__antithesis_instrumentation__.Notify(4)
	fmt.Println("Hello, world!")
	assert.Always(true, "always in main", nil)
	foo(true)
	foo(false)
}
