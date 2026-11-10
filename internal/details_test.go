//go:build enable_antithesis_sdk

package internal

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

// Details behaviour the cross-SDK conformance harness cannot reach: Go-native
// value types (typed nils, Marshalers, errors, every numeric width), cycles
// and shared subtrees, and caller data staying untouched. The wire shape and
// the diagnostics for JSON-shaped input are its job.

type customDetails string

func (v customDetails) MarshalJSON() ([]byte, error) { return []byte(v), nil }

type failedDetails struct{}

func (failedDetails) MarshalJSON() ([]byte, error) { return nil, errors.New("custom failure") }

type nilError struct{}

func (*nilError) Error() string { return "nil error" }

type customError struct{}

// A named integer reaches the wire as a number; one with MarshalText as a string.
type namedID uint64

type textID uint64

func (t textID) MarshalText() ([]byte, error) { return strconv.AppendUint(nil, uint64(t), 10), nil }

func (customError) Error() string                { return "error text" }
func (customError) MarshalJSON() ([]byte, error) { return []byte(`{"custom":true}`), nil }

// A struct exercising encoding/json's field rules.
type embedded struct {
	Inner float64 `json:"inner"`
}

type tagged struct {
	Kept    float64 `json:"kept"`
	Renamed int     `json:"renamed,omitempty"`
	Skipped int     `json:"-"`
	Quoted  float64 `json:"quoted,string"`
	hidden  int
	embedded
}

func TestDetailsShapes(t *testing.T) {
	var pointer *int
	for _, tt := range []struct {
		name  string
		value any
		want  string
	}{
		{"nil error", (*nilError)(nil), ""},
		{"nil slice", []any(nil), ""}, {"nil pointer", pointer, ""},
		{"custom null", customDetails(` null `), ""},
		{"custom object", customDetails(`{"x":1}`), `{"x":1}`},
		{"custom scalar", customDetails(`false`), `{"value":false}`},
		{"raw", json.RawMessage(`[1]`), `{"value":[1]}`},
		{"nested nil", map[string]any{"m": map[string]any(nil), "s": []any(nil)}, `{"m":null,"s":null}`},
		{"error", map[string]any{"e": errors.New("broken")}, `{"e":"broken"}`},
		{"custom error", map[string]any{"e": customError{}}, `{"e":{"custom":true}}`},
		{"float32 non-finite", map[string]any{"f": float32(math.Inf(-1))}, `{"f":"-Infinity"}`},
		{"integers", []any{json.Number("9007199254740993"), uint64(1 << 63), int8(-128)}, `{"value":[9007199254740993,9223372036854775808,-128]}`},
		// Normalization preserves struct field order and encoding tags.
		{"struct untouched", tagged{Kept: 1, Skipped: 5, Quoted: 1.5, hidden: 7, embedded: embedded{Inner: 2}}, `{"kept":1,"quoted":"1.5","inner":2}`},
		{"struct normalized", &tagged{Kept: math.NaN(), Skipped: 5, Quoted: 1.5, hidden: 7, embedded: embedded{Inner: 2}}, `{"kept":"NaN","quoted":"1.5","inner":2}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(Details(tt.value, "shape")); got != tt.want {
				t.Fatalf("got %s, want %s", got, tt.want)
			}
		})
	}
}

// Redirect SDK output to a file and return a reader of its lines.
func captureOutput(t *testing.T) func() []string {
	for _, warned := range []*atomic.Bool{&warnedNonFinite, &warnedUnsafeInteger, &warnedNonFiniteOperand} {
		previous := warned.Swap(false)
		t.Cleanup(func() { warned.Store(previous) })
	}
	old := handler
	file, err := os.CreateTemp(t.TempDir(), "output")
	if err != nil {
		t.Fatal(err)
	}
	handler = &localHandler{file}
	t.Cleanup(func() { handler = old; file.Close() })
	return func() []string {
		data, err := os.ReadFile(file.Name())
		if err != nil {
			t.Fatal(err)
		}
		if len(data) == 0 {
			return nil
		}
		return strings.Split(strings.TrimSpace(string(data)), "\n")
	}
}

func TestDetailsFailures(t *testing.T) {
	output := captureOutput(t)
	cycle := map[string]any{}
	cycle["self"] = cycle
	// The NaN is visited before the cycle and must not add a warning: a
	// failure reports only the error.
	slice := []any{math.NaN(), nil}
	slice[1] = slice
	for _, value := range []any{cycle, slice, customDetails(`invalid`), failedDetails{}} {
		if got := Details(value, "failure context"); got != nil {
			t.Fatalf("got %s", got)
		}
	}
	lines := output()
	if got := diagnosticCodes(t, lines, "failure context"); !slices.Equal(got, []int{6001, 6001, 6001, 6001}) {
		t.Fatalf("got %v", got)
	}
	if !strings.Contains(strings.Join(lines, "\n"), "custom failure") {
		t.Fatal("missing custom failure cause")
	}
}

// The codes of the captured diagnostics, each checked to carry context. The
// message text is pinned by the conformance harness, not here.
func diagnosticCodes(t *testing.T, lines []string, context string) []int {
	t.Helper()
	var codes []int
	for _, line := range lines {
		var record map[string]struct {
			Code    int
			Context string
		}
		if err := json.Unmarshal([]byte(line), &record); err != nil || len(record) != 1 {
			t.Fatalf("not a diagnostic: %s", line)
		}
		for _, diagnostic := range record {
			if diagnostic.Context != context {
				t.Fatalf("bad context: %s", line)
			}
			codes = append(codes, diagnostic.Code)
		}
	}
	return codes
}

func TestDetailsWarnings(t *testing.T) {
	for _, tt := range []struct {
		name  string
		value any
		codes []int
	}{
		{"float32", map[string]any{"f": float32(math.Inf(-1))}, []int{6002}},
		{"json.Number beyond int64", json.Number("-99999999999999999999"), []int{6003}},
		{"int64", int64(-1 << 53), []int{6003}},
		{"uint64", uint64(1 << 53), []int{6003}},
		{"named uint64", namedID(1 << 53), []int{6003}},
		{"inside struct and pointer", &tagged{Renamed: 1 << 53}, []int{6003}},
		// Safe integers of every native width, and fractional or exponent
		// json.Numbers, are silent even when their mantissa overflows int64.
		{"quiet", []any{int64(1<<53 - 1), -(1<<53 - 1), uint32(math.MaxUint32),
			json.Number("99999999999999999999e5"), json.Number("99999999999999999999.5"), textID(1 << 63)}, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(t)
			Details(tt.value, "warning context")
			if got := diagnosticCodes(t, output(), "warning context"); !slices.Equal(got, tt.codes) {
				t.Fatalf("got %v, want %v", got, tt.codes)
			}
		})
	}
}

func TestDetailsDoesNotMutateOrRejectSharedTrees(t *testing.T) {
	e := errors.New("broken")
	shared := map[string]any{"error": e}
	input := map[string]any{"a": shared, "b": shared}
	if got := string(Details(input, "shared")); got != `{"a":{"error":"broken"},"b":{"error":"broken"}}` {
		t.Fatal(got)
	}
	if shared["error"] != e {
		t.Fatal("mutated user map")
	}
}

func TestGeneratedFieldShadowsScalarWarning(t *testing.T) {
	lines := captureOutput(t)
	got := DetailsWithFields(math.NaN(), map[string]any{"value": true}, "shadowed")
	if string(got) != `{"value":true}` {
		t.Fatalf("got %s", got)
	}
	if output := lines(); len(output) != 0 {
		t.Fatalf("unexpected diagnostic: %v", output)
	}
}

// Each warning code is claimed once per process, across details and packets.
func TestDetailsWarningsClaimedOnce(t *testing.T) {
	output := captureOutput(t)
	Details(map[string]any{"n": uint64(1 << 60), "f": math.Inf(1)}, "first")
	Details(int64(-1<<60), "second")
	EmitWithDetails(map[string]any{"packet": map[string]any{"f": math.NaN(), "n": int64(1 << 53)}}, "third")
	var codes []int
	var packets int
	for _, line := range output() {
		if strings.HasPrefix(line, `{"packet":`) {
			packets++
			continue
		}
		codes = append(codes, diagnosticCodes(t, []string{line}, "first")...)
	}
	if !slices.Equal(codes, []int{6002, 6003}) || packets != 1 {
		t.Fatalf("got codes %v, %d packets", codes, packets)
	}
}

func TestEmitWithDetailsDropsFailedPackets(t *testing.T) {
	output := captureOutput(t)
	EmitWithDetails(map[string]any{"packet": map[string]any{"f": math.NaN(), "c": make(chan int)}}, "broken")
	EmitWithDetails(map[string]any{"packet": nil}, "kept")
	lines := output()
	if len(lines) != 2 || diagnosticCodes(t, lines[:1], "broken")[0] != 6001 || lines[1] != `{"packet":null}` {
		t.Fatalf("got %q", lines)
	}
}
