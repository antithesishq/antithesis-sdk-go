package random

import (
	"testing"
)

// To execute tests:
//
// go test -v github.com/antithesishq/antithesis-sdk-go/random

func TestRandomChoice(t *testing.T) {
	type Thing struct {
		chosenCount int
	}

	choices := []any{
		&Thing{},
		&Thing{},
		&Thing{},
		&Thing{},
		&Thing{},
	}

	const N = 100
	for i := 0; i < N; i++ {
		chosen := RandomChoice(choices)
		chosen.(*Thing).chosenCount += 1
	}

	for i, anything := range choices {
		thing := anything.(*Thing)
		t.Logf("Thing %d/%d was chosen %d times", i+1, len(choices), thing.chosenCount)
		if thing.chosenCount == 0 {
			t.Fatalf("Some element was never chosen in %d random choices!", N)
		}
	}
}

func TestChoiceGenericCompatibility(t *testing.T) {
	type Thing struct {
		chosenCount int
	}

	choices := []any{
		&Thing{},
		&Thing{},
		&Thing{},
		&Thing{},
		&Thing{},
	}

	const N = 100
	for i := 0; i < N; i++ {
		chosen := RandomChoiceG(choices)
		chosen.(*Thing).chosenCount += 1
	}

	for i, anything := range choices {
		thing := anything.(*Thing)
		t.Logf("Thing %d/%d was chosen %d times", i+1, len(choices), thing.chosenCount)
		if thing.chosenCount == 0 {
			t.Fatalf("Some element was never chosen in %d random choices!", N)
		}
	}
}

func TestChoiceGenericMixedArrayCompatibility(t *testing.T) {
	type This struct {
		thisCount int
	}

	type That struct {
		thatCount int
	}

	choices := []any{
		&This{},
		&This{},
		&That{},
		&That{},
		&That{},
	}

	const N = 100
	for i := 0; i < N; i++ {
		chosen := RandomChoiceG(choices)
		if t1, ok := chosen.(*This); ok {
			t1.thisCount += 1
		}
		if t2, ok := chosen.(*That); ok {
			t2.thatCount += 1
		}
	}

	for i, anything := range choices {
		if t1, ok := anything.(*This); ok {
			t.Logf("This at index %d of %d was chosen %d times", i+1, len(choices), t1.thisCount)
			if t1.thisCount == 0 {
				t.Fatalf("'This' element was never chosen in %d random choices!", N)
			}
		}

		if t2, ok := anything.(*That); ok {
			t.Logf("'That' at index %d of %d was chosen %d times", i+1, len(choices), t2.thatCount)
			if t2.thatCount == 0 {
				t.Fatalf("'That' element was never chosen in %d random choices!", N)
			}
		}
	}
}

func TestRandomChoiceGenericMixedPrimitives(t *testing.T) {

	choices := []any{
		"Hello",
		12.4,
		"How",
		true,
		"You",
		10025,
	}

	counts := make(map[any]int)
	const N = 100
	for i := 0; i < N; i++ {
		chosen := RandomChoiceG(choices)
		counts[chosen] += 1
	}

	for i, s := range choices {
		count, present := counts[s]
		t.Logf("Item %v %d/%d was chosen %d times", s, i+1, len(choices), count)
		if !present {
			t.Fatalf("Some element was never chosen in %d random choices!", N)
		}
	}
}

func TestRandomChoiceGeneric(t *testing.T) {

	choices := []string{
		"Hello",
		"World",
		"How",
		"Are",
		"You",
		"?",
	}

	counts := make(map[string]int)
	const N = 100
	for i := 0; i < N; i++ {
		chosen := RandomChoiceG(choices)
		counts[chosen] += 1
	}

	for i, s := range choices {
		count, present := counts[s]
		t.Logf("Item %d/%d was chosen %d times", i+1, len(choices), count)
		if !present {
			t.Fatalf("Some element was never chosen in %d random choices!", N)
		}
	}
}

func TestEmptyFloatChoiceGeneric(t *testing.T) {
	var choices []float64

	got := RandomChoiceG(choices)
	want := float64(0.0)
	if got != want {
		t.Fatalf("Unexpected choice received - got %v want %f", got, want)
	}
}

func TestEmptyStringChoiceGeneric(t *testing.T) {
	var choices []string

	got := RandomChoiceG(choices)
	want := ""
	if got != want {
		t.Fatalf("Unexpected choice received - got %v want %s", got, want)
	}
}
