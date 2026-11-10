//go:build !no_antithesis_sdk

package json

import (
	std "encoding/json"
	"errors"
	"math"
	"reflect"
	"testing"
)

type embedded struct{ X int }
type outer struct {
	embedded
	X   int
	Bad float64
}
type namedFloat float64

func (*namedFloat) MarshalJSON() ([]byte, error) { return []byte(`"custom"`), nil }

type omitted int

func (omitted) IsZero() bool { return true }

type byteError byte

func (byteError) Error() string { return "byte error" }

type dualError struct{}

func (dualError) Error() string                { return "error" }
func (dualError) MarshalText() ([]byte, error) { return []byte("text"), nil }

type jsonErrorValue struct{ dualError }

func (jsonErrorValue) MarshalJSON() ([]byte, error) { return []byte(`"json"`), nil }

type failing struct{}

func (failing) MarshalJSON() ([]byte, error) { return nil, errors.New("failure") }

type keyCallback struct{ Calls *int }

func (k keyCallback) MarshalText() ([]byte, error) { (*k.Calls)++; return []byte("left"), nil }

type unreachable struct{}

func (unreachable) IsZero() bool                 { panic("discarded IsZero called") }
func (unreachable) MarshalJSON() ([]byte, error) { panic("discarded marshaler called") }

func TestStandardSemantics(t *testing.T) {
	f := namedFloat(3)
	n := 42
	// Build the intentional tag collision at runtime so go vet can still catch
	// accidental duplicate tags in ordinary structs.
	ambiguous := reflect.New(reflect.StructOf([]reflect.StructField{
		{Name: "A", Type: reflect.TypeOf(0), Tag: `json:"same"`},
		{Name: "B", Type: reflect.TypeOf(0), Tag: `json:"same"`},
	})).Elem()
	ambiguous.Field(0).SetInt(1)
	ambiguous.Field(1).SetInt(2)
	values := []any{outer{embedded{2}, 1, 3}, f, &f, struct {
		N *int `json:",string"`
	}{&n}, map[int]string{2: "two"}, []byte{1, 2}, std.RawMessage(`{"a":1}`), struct {
		N omitted `json:",omitzero"`
	}{17}, ambiguous.Interface()}
	for _, v := range values {
		want, err := std.Marshal(v)
		got, w, e := MarshalDetails(v)
		if err != nil || e != nil || string(got) != string(want) || w != (Warnings{}) {
			t.Errorf("%T: want %s (%v), got %s (%v), warnings %+v", v, want, err, got, e, w)
		}
	}
}

func TestNumericDiagnostics(t *testing.T) {
	cases := []struct {
		name     string
		value    any
		want     string
		warnings Warnings
	}{
		{"dominance", outer{embedded{2}, 1, math.NaN()}, `{"X":1,"Bad":"NaN"}`, Warnings{NonFinite: true}},
		{"infinity", []float64{math.Inf(1), math.Inf(-1)}, `["Infinity","-Infinity"]`, Warnings{NonFinite: true}},
		{"integer", int64(1 << 53), `9007199254740992`, Warnings{UnsafeInteger: true}},
		{"negative", int64(-(1 << 53)), `-9007199254740992`, Warnings{UnsafeInteger: true}},
		{"unsigned", uint64(1 << 63), `9223372036854775808`, Warnings{UnsafeInteger: true}},
		{"safe", int64((1 << 53) - 1), `9007199254740991`, Warnings{}},
		{"float", float64(1 << 53), `9007199254740992`, Warnings{}},
		{"number", std.Number("9007199254740992"), `9007199254740992`, Warnings{UnsafeInteger: true}},
		{"huge number", std.Number("999999999999999999999999999999999999"), `999999999999999999999999999999999999`, Warnings{UnsafeInteger: true}},
		{"fraction number", std.Number("9007199254740992.0"), `9007199254740992.0`, Warnings{}},
		{"exponent number", std.Number("1e100"), `1e100`, Warnings{}},
		{"quoted", struct {
			N int64 `json:",string"`
		}{1 << 53}, `{"N":"9007199254740992"}`, Warnings{}},
		{"quoted number", struct {
			N std.Number `json:",string"`
		}{"9007199254740992"}, `{"N":"9007199254740992"}`, Warnings{}},
		{"map key", map[int64]int{1 << 53: 0}, `{"9007199254740992":0}`, Warnings{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, w, e := MarshalDetails(c.value)
			if e != nil || string(got) != c.want || w != c.warnings {
				t.Fatalf("got %s, %+v, %v; want %s, %+v", got, w, e, c.want, c.warnings)
			}
		})
	}
}

func TestMethodPrecedence(t *testing.T) {
	for _, c := range []struct {
		v    any
		want string
	}{{errors.New("message"), `"message"`},
		{[]byteError{1}, `["byte error"]`}, {dualError{}, `"error"`}, {jsonErrorValue{}, `"json"`}, {struct{ E error }{jsonErrorValue{}}, `{"E":"json"}`}} {
		got, w, e := MarshalDetails(c.v)
		if e != nil || string(got) != c.want || w != (Warnings{}) {
			t.Fatal(string(got), w, e)
		}
	}
	f := namedFloat(math.NaN())
	for _, v := range []any{&f, []namedFloat{f}} {
		_, w, e := MarshalDetails(v)
		if e != nil || w.NonFinite {
			t.Fatal(w, e)
		}
	}
	for _, v := range []any{f, map[string]namedFloat{"f": f}} {
		_, w, e := MarshalDetails(v)
		if e != nil || !w.NonFinite {
			t.Fatal(w, e)
		}
	}
}

func TestFailureDiscardsWarnings(t *testing.T) {
	cyclic := map[string]any{}
	cyclic["self"] = cyclic
	for _, v := range []any{[]any{math.NaN(), failing{}}, []any{int64(1 << 53), make(chan int)}, std.Number("NaN"), cyclic} {
		got, w, e := MarshalDetails(v)
		if e == nil || got != nil || w != (Warnings{}) {
			t.Fatalf("%T: %s %+v %v", v, got, w, e)
		}
	}
	_, w, e := MarshalDetails(1)
	if e != nil || w != (Warnings{}) {
		t.Fatal(w, e)
	}
}

func TestExcludedRootFields(t *testing.T) {
	fields := map[string]any{"left": 42}
	for _, v := range []any{map[string]any{"left": unreachable{}, "nested": map[string]int{"left": 7}}, struct {
		Left   unreachable    `json:"left,omitzero"`
		Nested map[string]int `json:"nested"`
	}{Nested: map[string]int{"left": 7}}} {
		got, w, e := MarshalDetailsWithFields(v, fields)
		if e != nil || string(got) != `{"nested":{"left":7},"left":42}` || w != (Warnings{}) {
			t.Fatal(string(got), w, e)
		}
	}
	got, w, e := MarshalDetailsWithFields(map[string]any{"left": math.NaN()}, fields)
	if e != nil || string(got) != `{"left":42}` || w != (Warnings{}) {
		t.Fatal(string(got), w, e)
	}
	got, _, e = MarshalDetailsWithFields([]any{map[string]int{"left": 7}}, fields)
	if e != nil || string(got) != `{"value":[{"left":7}],"left":42}` {
		t.Fatal(string(got), e)
	}
	got, _, e = MarshalDetails(map[string]int{"left": 7})
	if e != nil || string(got) != `{"left":7}` {
		t.Fatal(string(got), e)
	}
}

var benchmarkValue = struct {
	Name  string
	Count int
	Data  []float64
}{"example", 42, []float64{1, 2, 3, 4}}
var benchmarkResult []byte

func BenchmarkStandard(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchmarkResult, _ = std.Marshal(benchmarkValue)
	}
}
func BenchmarkDetails(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchmarkResult, _, _ = MarshalDetails(benchmarkValue)
	}
}

func TestExclusionBoundaries(t *testing.T) {
	fields := map[string]any{"left": 1, "2": 1}
	var nilRoot *struct{ Left int }
	got, w, e := MarshalDetailsWithFields(nilRoot, fields)
	if e != nil || string(got) != `{"2":1,"left":1}` || w != (Warnings{}) {
		t.Fatal(string(got), w, e)
	}
	got, _, e = MarshalDetailsWithFields(map[int]any{2: unreachable{}, 3: 4}, fields)
	if e != nil || string(got) != `{"3":4,"2":1,"left":1}` {
		t.Fatal(string(got), e)
	}
	calls := 0
	got, _, e = MarshalDetailsWithFields(map[keyCallback]any{{&calls}: unreachable{}}, fields)
	if e != nil || string(got) != `{"2":1,"left":1}` || calls != 1 {
		t.Fatal(string(got), e, calls)
	}
	// Opaque output is merged after the fact: its marshaler runs once and its
	// colliding keys give way.
	got, _, e = MarshalDetailsWithFields(std.RawMessage(`{"left":7,"kept":[1]}`), fields)
	if e != nil || string(got) != `{"kept":[1],"2":1,"left":1}` {
		t.Fatal(string(got), e)
	}
}

// Fields are merged in the same traversal for every root shape.
func TestRootFieldShapes(t *testing.T) {
	fields := map[string]any{"left": 1, "right": namedFloat(0)}
	nested := map[string]any{"m": map[string]any{"m": 2, "value": 3}}
	for _, c := range []struct {
		name   string
		v      any
		fields map[string]any
		want   string
		w      Warnings
	}{
		{"nil root", nil, fields, `{"left":1,"right":0}`, Warnings{}},
		{"null marshaler", std.RawMessage(` null `), fields, `{"left":1,"right":0}`, Warnings{}},
		{"scalar", 5, fields, `{"value":5,"left":1,"right":0}`, Warnings{}},
		{"scalar marshaler", std.RawMessage(`7`), fields, `{"value":7,"left":1,"right":0}`, Warnings{}},
		{"non-finite scalar", math.NaN(), fields, `{"value":"NaN","left":1,"right":0}`, Warnings{NonFinite: true}},
		{"shadowed scalar", math.NaN(), map[string]any{"value": true}, `{"value":true}`, Warnings{}},
		{"shadowed array", []float64{math.Inf(1)}, map[string]any{"value": nil, "a": 1}, `{"a":1,"value":null}`, Warnings{}},
		{"field warnings", map[string]any{"a": 1}, map[string]any{"left": int64(1 << 53)}, `{"a":1,"left":9007199254740992}`, Warnings{UnsafeInteger: true}},
		{"field warnings on wrapped", 5, map[string]any{"f": math.Inf(-1)}, `{"value":5,"f":"-Infinity"}`, Warnings{NonFinite: true}},
		{"nested field keeps its keys", map[string]any{"m": 1}, nested, `{"m":{"m":2,"value":3}}`, Warnings{}},
		{"nested field under wrapped root", 5, nested, `{"value":5,"m":{"m":2,"value":3}}`, Warnings{}},
		{"nested field under null root", nil, nested, `{"m":{"m":2,"value":3}}`, Warnings{}},
		{"empty fields", map[string]any{"a": 1}, map[string]any{}, `{"a":1}`, Warnings{}},
		{"empty fields on scalar", 5, map[string]any{}, `{"value":5}`, Warnings{}},
		{"empty fields on nil", nil, map[string]any{}, `{}`, Warnings{}},
		{"empty struct", struct{}{}, fields, `{"left":1,"right":0}`, Warnings{}},
		{"html in key", map[string]any{"a": 1}, map[string]any{"<k>": "<v>"}, `{"a":1,"\u003ck\u003e":"\u003cv\u003e"}`, Warnings{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, w, e := MarshalDetailsWithFields(c.v, c.fields)
			if e != nil || string(got) != c.want || w != c.w {
				t.Fatalf("got %s, %+v, %v; want %s, %+v", got, w, e, c.want, c.w)
			}
		})
	}
}

func TestRootFieldFailures(t *testing.T) {
	broken := map[string]any{"left": make(chan int)}
	cyclic := map[string]any{}
	cyclic["self"] = cyclic
	for _, c := range []struct {
		v      any
		fields map[string]any
	}{{math.NaN(), broken}, {map[string]any{"a": math.NaN()}, broken}, {nil, broken}, {std.RawMessage(`{"a":1}`), broken},
		{cyclic, map[string]any{"left": 1}}, {map[string]any{"a": failing{}}, map[string]any{"left": 1}}, {5, map[string]any{"left": failing{}}}} {
		got, w, e := MarshalDetailsWithFields(c.v, c.fields)
		if e == nil || got != nil || w != (Warnings{}) {
			t.Fatalf("%T: %s %+v %v", c.v, got, w, e)
		}
	}
	// A failed encode leaves nothing behind for the next pooled use.
	got, w, e := MarshalDetails(map[string]any{"left": 7})
	if e != nil || string(got) != `{"left":7}` || w != (Warnings{}) {
		t.Fatal(string(got), w, e)
	}
}
