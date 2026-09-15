//go:build enable_antithesis_sdk

package assert

import "testing"

// Enabled must remain a constant: "if assert.Enabled" only disappears from a
// build without the SDK because the compiler can fold a constant condition.
// Using it in a constant declaration fails to build if it ever becomes a var.
const _ = Enabled

func TestEnabledIsTrueWithTheBuildTag(t *testing.T) {
	if !Enabled {
		t.Fatal("Enabled is false in a build with the enable_antithesis_sdk tag")
	}
}
