//go:build !no_antithesis_sdk && !race

package assert

import (
	"fmt"
	"testing"
)

// Race instrumentation can randomly discard pooled JSON buffers, so allocation
// comparisons run only in ordinary builds.
func largeDetails() map[string]any {
	details := make(map[string]any, 512)
	for i := 0; i < 512; i++ {
		details[fmt.Sprintf("field-%d", i)] = i
	}
	return details
}

func TestGuidanceOnlyDoesNotCopyDetails(t *testing.T) {
	details := largeDetails()
	t.Run("numeric", func(t *testing.T) {
		measure := func(id string, details map[string]any) float64 {
			value := 1000
			SometimesGreaterThan(value, 0, id, details)
			return testing.AllocsPerRun(100, func() { value++; SometimesGreaterThan(value, 0, id, details) })
		}
		without := measure(t.Name()+"/nil", nil)
		with := measure(t.Name()+"/map", details)
		if with != without {
			t.Fatalf("guidance-only allocations depend on details: nil=%g, map=%g", without, with)
		}
	})
	t.Run("boolean", func(t *testing.T) {
		values := []NamedBool{{"p", true}, {"q", false}}
		measure := func(id string, details map[string]any) float64 {
			AlwaysSome(values, id, details)
			return testing.AllocsPerRun(100, func() { AlwaysSome(values, id, details) })
		}
		without := measure(t.Name()+"/nil", nil)
		with := measure(t.Name()+"/map", details)
		if with != without {
			t.Fatalf("guidance-only allocations depend on details: nil=%g, map=%g", without, with)
		}
	})
}

func BenchmarkGuidanceOnlyDetails(b *testing.B) {
	details := largeDetails()
	for _, tc := range []struct {
		name    string
		details map[string]any
	}{{"Nil", nil}, {"LargeMap", details}} {
		b.Run("Numeric/"+tc.name, func(b *testing.B) {
			id := b.Name()
			numericGuidanceTracker.Delete(id)
			SometimesGreaterThan(1000, 0, id, tc.details)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				SometimesGreaterThan(i+1001, 0, id, tc.details)
			}
		})
		b.Run("Boolean/"+tc.name, func(b *testing.B) {
			values := []NamedBool{{"p", true}, {"q", false}}
			AlwaysSome(values, b.Name(), tc.details)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				AlwaysSome(values, b.Name(), tc.details)
			}
		})
	}
}
