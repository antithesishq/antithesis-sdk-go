//go:build enable_antithesis_sdk

package assert

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestSuppressedRawDoesNotAllocate(t *testing.T) {
	for _, condition := range []bool{true, false} {
		for _, details := range []map[string]any{nil, {"value": 42}} {
			id := t.Name()
			assertTracker.Delete(id)
			call := func() {
				AssertRaw(condition, "distinct message", details, "class", "function", "file.go", 3, 4, true, true, "always", "Always", id)
			}
			call()
			if allocs := testing.AllocsPerRun(100, call); allocs != 0 {
				t.Fatalf("condition=%t details=%v: %g allocations per suppressed call", condition, details, allocs)
			}
		}
	}
}

func TestRawGateUsesIDAndPreservesDeclarations(t *testing.T) {
	const childVariable = "ANTITHESIS_RAW_GATE_TEST_CHILD"
	if os.Getenv(childVariable) == "1" {
		for _, id := range []string{"first id", "second id"} {
			for _, condition := range []bool{true, false} {
				for _, hit := range []bool{true, true, false, false} {
					AssertRaw(condition, "shared message", nil, "", "", "", 0, 0, hit, true, "always", "Always", id)
				}
			}
		}
		return
	}
	path := t.TempDir() + "/events.jsonl"
	child := exec.Command(os.Args[0], "-test.run=^TestRawGateUsesIDAndPreservesDeclarations$")
	child.Env = append(os.Environ(), childVariable+"=1", "ANTITHESIS_SDK_LOCAL_OUTPUT="+path)
	if out, err := child.CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	type outcome struct {
		ID        string
		Condition bool
		Hit       bool
	}
	counts := map[outcome]int{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var event struct {
			Assertion *struct {
				ID        string `json:"id"`
				Message   string `json:"message"`
				Condition bool   `json:"condition"`
				Hit       bool   `json:"hit"`
			} `json:"antithesis_assert"`
		}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatal(err)
		}
		if a := event.Assertion; a != nil {
			if a.Message != "shared message" {
				continue
			}
			if a.ID != "first id" && a.ID != "second id" {
				t.Fatalf("unexpected assertion: %s", line)
			}
			counts[outcome{a.ID, a.Condition, a.Hit}]++
		}
	}
	for _, id := range []string{"first id", "second id"} {
		for _, condition := range []bool{true, false} {
			for hit, want := range map[bool]int{true: 1, false: 2} {
				key := outcome{id, condition, hit}
				if got := counts[key]; got != want {
					t.Errorf("%+v: got %d records, want %d", key, got, want)
				}
			}
		}
	}
}

func TestEmissionFailureRemainsRetryable(t *testing.T) {
	var tracker trackerInfo
	calls := 0
	emit := func() error {
		calls++
		return nil
	}
	tracker.emit(false, true, emit)
	if calls != 1 || tracker.passEmitted.Load() || tracker.failEmitted.Load() {
		t.Fatal("declaration changed hit tracking")
	}
	tracker.emit(true, true, func() error { return errors.New("output failure") })
	if tracker.passEmitted.Load() {
		t.Fatal("failed emission marked outcome emitted")
	}
	tracker.emit(true, true, emit)
	tracker.emit(true, true, emit)
	if calls != 2 || !tracker.passEmitted.Load() {
		t.Fatalf("retry produced %d successful emissions; want declaration and one hit", calls)
	}
}

func TestSuppressedAssertionRecordDoesNotAllocate(t *testing.T) {
	loc := &locationInfo{Filename: "file.go", Classname: "class"}
	details := map[string]any{"value": 42}
	for _, condition := range []bool{true, false} {
		assertTracker.Delete(t.Name())
		call := func() {
			assertImpl(condition, "distinct message", details, loc, true, true, "always", "Always", t.Name(), nil)
		}
		call()
		if allocs := testing.AllocsPerRun(100, call); allocs != 0 {
			t.Fatalf("condition=%t: %g allocations per suppressed assertion", condition, allocs)
		}
	}
}
