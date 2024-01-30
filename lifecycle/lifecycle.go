package lifecycle

import (
	"github.com/antithesishq/antithesis-sdk-go/internal"
)

// SetupComplet indicates that the system under test
// is ready for testing.
func SetupComplete() {
	internal.Json_data(map[string]string{"setup_status": "complete"})
}
