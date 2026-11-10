//go:build enable_antithesis_sdk

package assert

import (
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"testing"
)

type failedDetails struct{}

func (failedDetails) MarshalJSON() ([]byte, error) { return nil, errors.New("details failed") }

func TestRichDetailsPreserveInputAndOverrideFields(t *testing.T) {
	input := map[string]any{"left": failedDetails{}, "right": math.NaN(), "p": "old", "extra": 7}
	for _, tt := range []struct {
		name    string
		details map[string]any
		fields  map[string]any
		want    string
	}{
		{"map", input, numericFields(3, 2), `{"left":3,"right":2,"p":"old","extra":7}`},
		{"boolean", input, booleanFields([]NamedBool{{"left", true}, {"right", false}, {"p", true}, {"p", false}}), `{"left":true,"right":false,"p":false,"extra":7}`},
		{"struct", map[string]any{"nested": struct {
			Context int     `json:"context"`
			Value   float64 `json:"value"`
		}{7, math.NaN()}}, numericFields(3, 2), `{"nested":{"context":7,"value":"NaN"},"left":3,"right":2}`},
		{"pointer", map[string]any{"nested": &struct {
			Context int `json:"context"`
		}{7}}, booleanFields([]NamedBool{{"p", true}}), `{"nested":{"context":7},"p":true}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := assertInfo{Details: tt.details, fields: tt.fields, Hit: true, Id: t.Name()}
			got := f.event().A.Details
			var actual, expected any
			if err := json.Unmarshal(got, &actual); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(tt.want), &expected); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(actual, expected) {
				t.Fatalf("got %s, want %s", got, tt.want)
			}
		})
	}
	if _, ok := input["left"].(failedDetails); !ok || !math.IsNaN(input["right"].(float64)) || input["p"] != "old" {
		t.Fatal("mutated input", input)
	}
}

type countedDetails struct{ calls *int }

func (d countedDetails) MarshalJSON() ([]byte, error) {
	(*d.calls)++
	return []byte(`{"context":7}`), nil
}

func TestRichDetailsSerializationStaysBehindEmissionGate(t *testing.T) {
	assertTracker.Delete(t.Name())
	numericGuidanceTracker.Delete(t.Name())
	calls := 0
	details := map[string]any{"nested": countedDetails{&calls}}
	for i := 1; i <= 4; i++ {
		SometimesGreaterThan(i, 0, t.Name(), details)
	}
	if calls != 1 {
		t.Fatalf("serialized details %d times for one emitted outcome", calls)
	}
}
