//go:build !no_antithesis_sdk

package assert

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
)

type namedInt int
type namedInt8 int8
type namedInt16 int16
type namedInt32 int32
type namedInt64 int64
type namedUint uint
type namedUint8 uint8
type namedUint16 uint16
type namedUint32 uint32
type namedUint64 uint64
type namedUintptr uintptr
type namedFloat32 float32
type namedFloat64 float64

func checkNumericAllocations[T Number](t *testing.T, left, right T) {
	t.Helper()
	id := t.Name()
	AlwaysGreaterThan(left, right, id, nil)
	if got := testing.AllocsPerRun(100, func() { AlwaysGreaterThan(left, right, id, nil) }); got != 0 {
		t.Fatalf("suppressed rich assertion allocated %g times", got)
	}
	rawID := id + "/raw"
	raw := func() { NumericGuidanceRaw(left, right, rawID, rawID, "", "", "", 0, 0, "minimize", true) }
	raw()
	if got := testing.AllocsPerRun(100, raw); got != 0 {
		t.Fatalf("suppressed raw guidance allocated %g times", got)
	}
}

func TestSuppressedNumericDoesNotAllocate(t *testing.T) {
	t.Run("int", func(t *testing.T) { checkNumericAllocations(t, int64(1000), int64(999)) })
	t.Run("uint", func(t *testing.T) { checkNumericAllocations(t, uint64(1000), uint64(999)) })
	t.Run("float", func(t *testing.T) { checkNumericAllocations(t, 1000.5, 999.5) })
	t.Run("named int", func(t *testing.T) { checkNumericAllocations(t, namedInt64(1000), namedInt64(999)) })
	t.Run("named uint", func(t *testing.T) { checkNumericAllocations(t, namedUint64(1000), namedUint64(999)) })
	t.Run("named float", func(t *testing.T) { checkNumericAllocations(t, namedFloat64(1000.5), namedFloat64(999.5)) })
}

func emitNumericSequence[T Number](id, behavior string, pairs [][2]T) {
	NumericGuidanceRaw(T(0), T(0), id, id, "class", "function", "file", 7, 0, behavior, false)
	for _, pair := range pairs {
		NumericGuidanceRaw(pair[0], pair[1], id, id, "class", "function", "file", 7, 0, behavior, true)
	}
}

func emitNamedNumeric[T Number]() {
	var zero T
	emitNumericSequence(reflect.TypeOf(zero).Name(), "maximize", [][2]T{{0, 0}, {2, 0}, {1, 0}, {3, 0}, {3, 0}})
}

func TestNumericGuidanceRecords(t *testing.T) {
	if os.Getenv("ANTITHESIS_NUMERIC_TEST_CHILD") == "1" {
		emitNamedNumeric[namedInt]()
		emitNamedNumeric[namedInt8]()
		emitNamedNumeric[namedInt16]()
		emitNamedNumeric[namedInt32]()
		emitNamedNumeric[namedInt64]()
		emitNamedNumeric[namedUint]()
		emitNamedNumeric[namedUint8]()
		emitNamedNumeric[namedUint16]()
		emitNamedNumeric[namedUint32]()
		emitNamedNumeric[namedUint64]()
		emitNamedNumeric[namedUintptr]()
		emitNamedNumeric[namedFloat32]()
		emitNamedNumeric[namedFloat64]()
		emitNumericSequence("signed max", "maximize", [][2]int64{{0, 0}, {2, 0}, {1, 0}, {math.MaxInt64, math.MinInt64}, {math.MinInt64, math.MaxInt64}})
		emitNumericSequence("signed min", "minimize", [][2]int64{{0, 0}, {-2, 0}, {-1, 0}, {math.MinInt64, math.MaxInt64}, {math.MaxInt64, math.MinInt64}})
		emitNumericSequence("unsigned max", "maximize", [][2]uint64{{0, 0}, {2, 0}, {1, 0}, {math.MaxUint64, 0}, {0, math.MaxUint64}})
		emitNumericSequence("unsigned min", "minimize", [][2]uint64{{0, 0}, {0, 2}, {0, 1}, {0, math.MaxUint64}, {math.MaxUint64, 0}})
		emitNumericSequence("unsigned max first extreme", "maximize", [][2]uint64{{0, math.MaxUint64}, {0, 0}})
		emitNumericSequence("unsigned min first extreme", "minimize", [][2]uint64{{math.MaxUint64, 0}, {0, 0}})
		var workers sync.WaitGroup
		for worker := 0; worker < 8; worker++ {
			workers.Add(1)
			go func(worker int) {
				defer workers.Done()
				for i := 0; i < 32; i++ {
					value := worker*32 + i
					NumericGuidanceRaw(value, 0, "concurrent", "concurrent", "", "", "", 0, 0, "maximize", true)
				}
			}(worker)
		}
		workers.Wait()
		emitNumericSequence("fractional", "maximize", [][2]namedFloat64{{0, 0}, {0.25, 0}, {0.125, 0}, {0.5, 0}})
		return
	}
	path := t.TempDir() + "/events.jsonl"
	child := exec.Command(os.Args[0], "-test.run=^TestNumericGuidanceRecords$")
	child.Env = append(os.Environ(), "ANTITHESIS_NUMERIC_TEST_CHILD=1", "ANTITHESIS_SDK_LOCAL_OUTPUT="+path)
	if out, err := child.CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	hits := map[string][][2]string{}
	declarations := map[string]int{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var event struct {
			Guidance *struct {
				Message string          `json:"message"`
				Hit     bool            `json:"hit"`
				Data    json.RawMessage `json:"guidance_data"`
			} `json:"antithesis_guidance"`
		}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatal(err)
		}
		if event.Guidance == nil {
			continue
		}
		g := event.Guidance
		if !g.Hit {
			declarations[g.Message]++
			if g.Data != nil {
				t.Fatalf("declaration has operands: %s", line)
			}
			continue
		}
		var operands struct {
			Left  json.Number
			Right json.Number
		}
		if err := json.Unmarshal(g.Data, &operands); err != nil {
			t.Fatal(err)
		}
		hits[g.Message] = append(hits[g.Message], [2]string{string(operands.Left), string(operands.Right)})
	}
	expected := map[string][][2]string{
		"signed max":   {{"0", "0"}, {"2", "0"}, {"9223372036854775807", "-9223372036854775808"}},
		"signed min":   {{"0", "0"}, {"-2", "0"}, {"-9223372036854775808", "9223372036854775807"}},
		"unsigned max": {{"0", "0"}, {"2", "0"}, {"18446744073709551615", "0"}},
		"unsigned min": {{"0", "0"}, {"0", "2"}, {"0", "18446744073709551615"}},
		"fractional":   {{"0", "0"}, {"0.25", "0"}, {"0.5", "0"}},
		// Both elements emit: the first because nothing had been seen, the
		// second because it improves on it.
		"unsigned max first extreme": {{"0", "18446744073709551615"}, {"0", "0"}},
		"unsigned min first extreme": {{"18446744073709551615", "0"}, {"0", "0"}},
	}
	for _, name := range []string{"namedInt", "namedInt8", "namedInt16", "namedInt32", "namedInt64", "namedUint", "namedUint8", "namedUint16", "namedUint32", "namedUint64", "namedUintptr", "namedFloat32", "namedFloat64"} {
		expected[name] = [][2]string{{"0", "0"}, {"2", "0"}, {"3", "0"}}
	}
	concurrent := hits["concurrent"]
	delete(hits, "concurrent")
	previous := -1
	for _, operands := range concurrent {
		value, err := strconv.Atoi(operands[0])
		if err != nil || value <= previous || operands[1] != "0" {
			t.Fatalf("non-monotonic concurrent guidance: %v", concurrent)
		}
		previous = value
	}
	if previous != 255 {
		t.Fatalf("final concurrent maximum = %d, want 255", previous)
	}
	if !reflect.DeepEqual(hits, expected) {
		t.Fatalf("got emissions %v; want %v", hits, expected)
	}
	for id := range expected {
		if declarations[id] != 1 {
			t.Fatalf("%s: got %d declarations", id, declarations[id])
		}
	}
}

// Each scenario runs in its own process because 6004 is claimed once per
// process. Every finite hit after a non-finite one must still emit, which shows
// the skipped hit left the watermark alone. Only the rich assertions are driven
// here: the raw guidance entry point is the conformance harness's.
func TestNonFiniteOperandWarnsOncePerProcess(t *testing.T) {
	const childVariable = "ANTITHESIS_NON_FINITE_TEST_SCENARIO"
	rich := func(id string, lefts ...float64) {
		for _, left := range lefts {
			SometimesGreaterThan(left, 0, id, nil)
		}
	}
	scenarios := []struct {
		name string
		run  func(id string)
		want []string
	}{
		{"rich first hit", func(id string) { rich(id, math.Inf(1), math.NaN(), 1, math.Inf(-1), 2) },
			[]string{"warning 6002", "assert true", "warning 6004", "assert false", "guidance 1", "guidance 2"}},
		{"rich settled probe", func(id string) { rich(id, 1, math.Inf(1), math.NaN(), 2) },
			[]string{"assert true", "guidance 1", "warning 6004", "warning 6002", "assert false", "guidance 2"}},
	}
	if name := os.Getenv(childVariable); name != "" {
		for _, scenario := range scenarios {
			if scenario.name == name {
				scenario.run(name)
			}
		}
		return
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			path := t.TempDir() + "/events.jsonl"
			child := exec.Command(os.Args[0], "-test.run=^TestNonFiniteOperandWarnsOncePerProcess$")
			child.Env = append(os.Environ(), childVariable+"="+scenario.name, "ANTITHESIS_SDK_LOCAL_OUTPUT="+path)
			if out, err := child.CombinedOutput(); err != nil {
				t.Fatalf("%v: %s", err, out)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
				var event struct {
					Assertion *struct {
						Message   string
						Condition bool
					} `json:"antithesis_assert"`
					Guidance *struct {
						Message string
						Data    *struct{ Left json.Number } `json:"guidance_data"`
					} `json:"antithesis_guidance"`
					Warning *struct {
						Code    int
						Context string
					} `json:"antithesis_warning"`
				}
				if err := json.Unmarshal([]byte(line), &event); err != nil {
					t.Fatal(err)
				}
				switch {
				case event.Warning != nil:
					if event.Warning.Context != scenario.name {
						t.Fatalf("unexpected warning: %s", line)
					}
					got = append(got, fmt.Sprintf("warning %d", event.Warning.Code))
				case event.Assertion != nil && event.Assertion.Message == scenario.name:
					got = append(got, fmt.Sprintf("assert %t", event.Assertion.Condition))
				case event.Guidance != nil && event.Guidance.Message == scenario.name:
					got = append(got, "guidance "+string(event.Guidance.Data.Left))
				}
			}
			if !slices.Equal(got, scenario.want) {
				t.Fatalf("got %q, want %q", got, scenario.want)
			}
		})
	}
}
