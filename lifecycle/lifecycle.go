//go:build antithesis_sdk

// This package is part of the [Antithesis Go SDK], which enables Go applications to integrate with the [Antithesis platform].
//
// The lifecycle package contains functions which inform the Antithesis environment that particular test phases or milestones have been reached.
//
// [Antithesis Go SDK]: https://antithesis.com/docs/using_antithesis/sdk/go_sdk.html
// [Antithesis platform]: https://antithesis.com
package lifecycle

import (
	"github.com/antithesishq/antithesis-sdk-go/internal"
)

// Call this function when your system and workload are fully initialized. After this function is called, the Antithesis environment will take a snapshot of your system and begin [injecting faults].
//
// Calling this function multiple times, or from multiple processes, will have no effect. Antithesis will treat the first time any process called this function as the moment that the setup was completed.
//
// [injecting faults]: https://antithesis.com/docs/applications/reliability/fault_injection.html
func SetupComplete(details any) {
	statusBlock := map[string]any{
		"status":  "complete",
		"details": details,
	}
	internal.Json_data(map[string]any{"antithesis_setup": statusBlock})
}

// Causes an event with the name and details provided,
// to be sent to the Fuzzer and Notebook
func SendEvent(eventName string, details any) {
	internal.Json_data(map[string]any{eventName: details})
}
