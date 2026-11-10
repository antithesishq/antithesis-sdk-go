//go:build enable_antithesis_sdk

// Package lifecycle informs the Antithesis environment that particular test phases or milestones have been reached. It is part of the [Antithesis Go SDK], which enables Go applications to integrate with the [Antithesis platform].
//
// Both functions take the parameter details: Optional additional information provided by the user to add context for assertion failures. The information that is logged will appear in the logs section of a [triage report]. Normally the values passed to details are evaluated at runtime.
//
// Nil details (including values serialized as JSON null) are omitted for SetupComplete and become an empty object for SendEvent. Explicit empty objects are preserved; other JSON values are wrapped as {"value": v}.
//
// For example, SendEvent("progress", 42) emits {"progress": {"value": 42}}.
//
// Argument expressions always run. Existing maps are read synchronously and are not modified by the SDK. Serialization failures emit an antithesis_error containing the context and error message.
//
// [Antithesis Go SDK]: https://antithesis.com/docs/using_antithesis/sdk/go/
// [Antithesis platform]: https://antithesis.com
// [triage report]: https://antithesis.com/docs/reports/
package lifecycle

import (
	"encoding/json"
	"github.com/antithesishq/antithesis-sdk-go/internal"
)

// SetupComplete indicates to Antithesis that setup has completed. Call this function when your system and workload are fully initialized. After this function is called, Antithesis will take a snapshot of your system and begin [injecting faults].
//
// Calling this function multiple times or from multiple processes will have no effect. Antithesis will treat the first time any process called this function as the moment that the setup was completed.
//
// [injecting faults]: https://antithesis.com/docs/environment/fault_injection/
func SetupComplete(details any) {
	var event struct {
		Setup struct {
			Status  string          `json:"status"`
			Details json.RawMessage `json:"details,omitempty"`
		} `json:"antithesis_setup"`
	}
	event.Setup.Status = "complete"
	event.Setup.Details = internal.Details(details, "setup_complete")
	internal.Json_data(event)
}

// SendEvent indicates to Antithesis that a certain event has been reached. It provides greater information about the ordering of events during the course of testing in Antithesis.
//
// In addition to details, you also provide an eventName, which is the name of the event that you are logging. This name will appear in the logs section of a [triage report].
//
// [triage report]: https://antithesis.com/docs/reports/
func SendEvent(eventName string, details any) {
	data := internal.Details(details, eventName)
	if data == nil {
		data = json.RawMessage(`{}`)
	}
	internal.Json_data(map[string]json.RawMessage{eventName: data})
}
