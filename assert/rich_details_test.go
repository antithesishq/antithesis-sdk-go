//go:build enable_antithesis_sdk

package assert

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestRichDetailsBuiltOnlyForEmittedOutcomes(t *testing.T) {
	var tracker trackerInfo
	var builds atomic.Int32
	build := func() map[string]any {
		builds.Add(1)
		return map[string]any{"generated": true}
	}
	assertImpl(true, t.Name(), nil, &locationInfo{}, false, true, "always", "Always", t.Name(), build)
	if builds.Load() != 0 {
		t.Fatal("declaration built details")
	}
	var workers sync.WaitGroup
	for i := 0; i < 32; i++ {
		workers.Add(1)
		go func(condition bool) {
			defer workers.Done()
			for j := 0; j < 10; j++ {
				tracker.emit(true, condition, func() error { build(); return nil })
			}
		}(i%2 == 0)
	}
	workers.Wait()
	if builds.Load() != 2 {
		t.Fatalf("built details %d times for two emitted outcomes", builds.Load())
	}
}

func TestRichGuidanceKeepsCatalogLocation(t *testing.T) {
	if os.Getenv("ANTITHESIS_RICH_DETAILS_TEST_CHILD") == "1" {
		for _, id := range []string{"numeric location", "boolean location"} {
			AssertRaw(true, id, nil, "catalog class", "catalog function", "catalog.go", 1, 0, false, true, "always", "Always", id)
		}
		details := map[string]any{"left": "overridden", "p": "overridden", "context": 42}
		for i := 1; i <= 3; i++ {
			SometimesGreaterThan(i, 0, "numeric location", details)
			AlwaysSome([]NamedBool{{"p", true}}, "boolean location", details)
		}
		return
	}
	path := t.TempDir() + "/events.jsonl"
	child := exec.Command(os.Args[0], "-test.run=^TestRichGuidanceKeepsCatalogLocation$")
	child.Env = append(os.Environ(), "ANTITHESIS_RICH_DETAILS_TEST_CHILD=1", "ANTITHESIS_SDK_LOCAL_OUTPUT="+path)
	if out, err := child.CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	guidanceCounts := map[string]int{}
	assertionCounts := map[string]int{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var event map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"antithesis_assert", "antithesis_guidance"} {
			raw, ok := event[key]
			if !ok {
				continue
			}
			var record struct {
				Message  string
				Hit      bool
				Location struct {
					Class string
					File  string
				}
				Details map[string]any
			}
			if err := json.Unmarshal(raw, &record); err != nil {
				t.Fatal(err)
			}
			if record.Message != "numeric location" && record.Message != "boolean location" {
				continue
			}
			if !record.Hit {
				continue
			}
			if record.Location.Class != "catalog class" || record.Location.File != "catalog.go" {
				t.Fatalf("lost catalog location: %s", raw)
			}
			if key == "antithesis_guidance" {
				guidanceCounts[record.Message]++
				continue
			}
			assertionCounts[record.Message]++
			if record.Details["context"] != float64(42) {
				t.Fatalf("lost user details: %s", raw)
			}
			if record.Message == "numeric location" && (record.Details["left"] != float64(1) || record.Details["right"] != float64(0)) {
				t.Fatalf("lost numeric fields: %s", raw)
			}
			if record.Message == "boolean location" && record.Details["p"] != true {
				t.Fatalf("lost boolean fields: %s", raw)
			}
		}
	}
	for _, id := range []string{"numeric location", "boolean location"} {
		if guidanceCounts[id] != 3 || assertionCounts[id] != 1 {
			t.Fatalf("%s: guidance=%d assertions=%d", id, guidanceCounts[id], assertionCounts[id])
		}
	}
}
