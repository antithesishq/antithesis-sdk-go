package assert

import (
    "os"
    "testing"
    "github.com/antithesishq/antithesis-sdk-go/local"
)

var test_result bool 

func TestCanEmitLocally(t *testing.T) {
    os.Setenv(local.LocalOutputEnvVar,"abc")
    defer os.Unsetenv(local.LocalOutputEnvVar)
    can_emit := CanEmit()
    if !can_emit {
         t.Errorf("Unable to emit locally when %q is set", local.LocalOutputEnvVar)
    }
}

func TestWillNotEmitLocally(t *testing.T) {
    os.Unsetenv(local.LocalOutputEnvVar)
    no_emit := NoEmit()
    if !no_emit {
         t.Errorf("Able to emit locally when %q is not set", local.LocalOutputEnvVar)
    }
}

func BenchmarkAlways(b *testing.B) {
    os.Unsetenv(local.LocalOutputEnvVar)
    for i := 0; i < b.N; i++ {
        Always("statement", true, nil)
    }
}

func BenchmarkCanEmitWithLocalEmitDisabled(b *testing.B) {
    os.Unsetenv(local.LocalOutputEnvVar)
    result := false
    for i := 0; i < b.N; i++ {
        result = CanEmit()
    }
    test_result = result
}

func BenchmarkNoEmitWithLocalEmitDisabled(b *testing.B) {
    os.Unsetenv(local.LocalOutputEnvVar)
    result := false
    for i := 0; i < b.N; i++ {
        result = NoEmit()
    }
    test_result = result
}

func BenchmarkCanEmitWithLocalEmitEnabled(b *testing.B) {
    os.Setenv(local.LocalOutputEnvVar, "abc")
    result := false
    for i := 0; i < b.N; i++ {
        result = CanEmit()
    }
    os.Unsetenv(local.LocalOutputEnvVar)
    test_result = result
}

func BenchmarkNoEmitWithLocalEmitEnabled(b *testing.B) {
    os.Setenv(local.LocalOutputEnvVar, "abc")
    result := false
    for i := 0; i < b.N; i++ {
        result = NoEmit()
    }
    os.Unsetenv(local.LocalOutputEnvVar)
    test_result = result
}
