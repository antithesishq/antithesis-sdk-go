package internal

import (
	"os"
	"testing"
)

var test_result bool

func TestCanEmitLocally(t *testing.T) {
	os.Setenv(localOutputEnvVar, "abc")
	defer os.Unsetenv(localOutputEnvVar)
	if no_emit() {
		t.Errorf("Unable to emit locally when %q is set", localOutputEnvVar)
	}
}

func TestWillNotEmitLocally(t *testing.T) {
	os.Unsetenv(localOutputEnvVar)
	No_emit := no_emit()
	if !No_emit {
		t.Errorf("Able to emit locally when %q is not set", localOutputEnvVar)
	}
}

func BenchmarkNoEmitWithLocalEmitDisabled(b *testing.B) {
	os.Unsetenv(localOutputEnvVar)
	result := false
	for i := 0; i < b.N; i++ {
		result = no_emit()
	}
	test_result = result
}

func BenchmarkNoEmitWithLocalEmitEnabled(b *testing.B) {
	os.Setenv(localOutputEnvVar, "abc")
	result := false
	for i := 0; i < b.N; i++ {
		result = no_emit()
	}
	os.Unsetenv(localOutputEnvVar)
	test_result = result
}
