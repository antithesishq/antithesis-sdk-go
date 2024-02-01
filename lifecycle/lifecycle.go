// Package lifecycle allows callers control of the [Antithesis environment].
//
// Callers running outside of the Antithesis environment will see no effect
// from these calls. Optionally, calls will log their operations to the file
// pointed to in the environment variable ANTITHESIS_SDK_LOCAL_OUTPUT, if
// it is defined.
//
// [Antithesis testing platform]: https://antithesis.com
package lifecycle

import (
	"github.com/antithesishq/antithesis-sdk-go/internal"
)

// To be called once system setup is complete and the system is ready for
// exploration.
func SetupComplete() {
	internal.Json_data(map[string]string{"setup_status": "complete"})
}
