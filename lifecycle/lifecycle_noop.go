//go:build !antithesis_sdk

package lifecycle

func SetupComplete(details any)               {}
func SendEvent(eventName string, details any) {}
