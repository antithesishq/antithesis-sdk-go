//go:build enable_antithesis_sdk

package assert

import "testing"

// Warm the trackers outside the measurement. No local output file is needed:
// the fallback handler still serializes events before discarding their bytes.
func BenchmarkSuppressedAssertion(b *testing.B) {
	b.Run("Nil", func(b *testing.B) {
		Always(true, b.Name(), nil)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			Always(true, b.Name(), nil)
		}
	})
	b.Run("ReusedMap", func(b *testing.B) {
		details := map[string]any{"count": 1000}
		Always(true, b.Name(), details)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			Always(true, b.Name(), details)
		}
	})
	b.Run("StaticMapLiteral", func(b *testing.B) {
		Always(true, b.Name(), nil)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			Always(true, b.Name(), map[string]any{"count": 1000})
		}
	})
	b.Run("MapLiteral", func(b *testing.B) {
		Always(true, b.Name(), nil)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			Always(true, b.Name(), map[string]any{"count": i + 1000})
		}
	})
}

func BenchmarkSuppressedNumeric(b *testing.B) {
	b.Run("Int", func(b *testing.B) {
		AlwaysGreaterThan(1000, 999, b.Name(), nil)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			AlwaysGreaterThan(1000, 999, b.Name(), nil)
		}
	})
	b.Run("Float", func(b *testing.B) {
		AlwaysGreaterThan(1000.5, 999.5, b.Name(), nil)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			AlwaysGreaterThan(1000.5, 999.5, b.Name(), nil)
		}
	})
	b.Run("Raw", func(b *testing.B) {
		NumericGuidanceRaw(1000, 999, b.Name(), b.Name(), "", "", "", 0, 0, "minimize", true)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			NumericGuidanceRaw(1000, 999, b.Name(), b.Name(), "", "", "", 0, 0, "minimize", true)
		}
	})
}

func BenchmarkBooleanGuidance(b *testing.B) {
	values := []NamedBool{{"first", true}, {"second", false}}
	AlwaysSome(values, b.Name(), nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		AlwaysSome(values, b.Name(), nil)
	}
}

func BenchmarkEmittedAssertion(b *testing.B) {
	for _, tc := range []struct {
		name    string
		details map[string]any
	}{
		{"Nil", nil},
		{"Flat", map[string]any{"count": 1000}},
		{"Nested", map[string]any{"object": map[string]any{"count": 1000}, "array": []any{1000, "text"}}},
	} {
		b.Run(tc.name, func(b *testing.B) {
			Always(true, b.Name(), tc.details)
			entry := getTrackerEntry(b.Name(), "", "")
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Reuse the tracker to exclude registration growth and ID formatting.
				entry.passEmitted.Store(false)
				Always(true, b.Name(), tc.details)
			}
		})
	}
}
