package random

import (
	"testing"
)

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
