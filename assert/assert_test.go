package assert

import (
	"os"
	"testing"
)

func BenchmarkAlways(b *testing.B) {
	os.Unsetenv("ANTITHESIS_SDK_LOCAL_OUTPUT")
	for i := 0; i < b.N; i++ {
		Always("statement", true, nil)
	}
}
