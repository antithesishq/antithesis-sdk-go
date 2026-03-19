package assertions

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"testing"

	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
)

func init() {
	// Ensure the global logWriter is initialized for tests.
	common.NewLogWriter("", 0)
}

func absTestdata(fixture string) string {
	p, err := filepath.Abs(filepath.Join("testdata", fixture))
	if err != nil {
		panic(err)
	}
	return p
}

func TestSingleMain(t *testing.T) {
	dir := absTestdata("single_main")
	scanner := NewAssertionScanner(false, dir, dir)
	if err := scanner.ScanAll(); err != nil {
		t.Fatalf("ScanAll failed: %v", err)
	}

	bins := scanner.binaries
	if len(bins) != 1 {
		t.Fatalf("expected 1 binary, got %d", len(bins))
	}

	bc := bins[0]
	// single_main has 3 assertion calls: Always, Sometimes, Reachable
	if got := len(bc.expects); got != 3 {
		t.Errorf("expected 3 assertions, got %d", got)
	}

	msgs := collectMessages(bc.expects)
	if !slices.Contains(msgs, "always true") {
		t.Errorf("expected %q in %v", "always true", msgs)
	}
	if !slices.Contains(msgs, "sometimes true") {
		t.Errorf("expected %q in %v", "sometimes true", msgs)
	}
	if !slices.Contains(msgs, "reached main") {
		t.Errorf("expected %q in %v", "reached main", msgs)
	}
}

func TestMultiMain(t *testing.T) {
	dir := absTestdata("multi_main")
	scanner := NewAssertionScanner(false, dir, dir)
	if err := scanner.ScanAll(); err != nil {
		t.Fatalf("ScanAll failed: %v", err)
	}

	bins := scanner.binaries
	if len(bins) != 2 {
		t.Fatalf("expected 2 binaries, got %d", len(bins))
	}

	// Sort by RelDir for deterministic assertions
	sort.Slice(bins, func(i, j int) bool {
		return bins[i].relDir < bins[j].relDir
	})

	// cmd/a imports shared + aonly: 1 (a main) + 1 (shared) + 1 (aonly) = 3 assertions
	binA := bins[0]
	if binA.relDir != filepath.Join("cmd", "a") {
		t.Errorf("expected binary RelDir cmd/a, got %s", binA.relDir)
	}
	msgsA := collectMessages(binA.expects)
	if !slices.Contains(msgsA, "a main assertion") {
		t.Errorf("expected %q in %v", "a main assertion", msgsA)
	}
	if !slices.Contains(msgsA, "shared assertion") {
		t.Errorf("expected %q in %v", "shared assertion", msgsA)
	}
	if !slices.Contains(msgsA, "aonly assertion") {
		t.Errorf("expected %q in %v", "aonly assertion", msgsA)
	}
	if len(binA.expects) != 3 {
		t.Errorf("binary A: expected 3 assertions, got %d: %v", len(binA.expects), msgsA)
	}

	// cmd/b imports shared only: 1 (b main) + 1 (shared) = 2 assertions
	binB := bins[1]
	if binB.relDir != filepath.Join("cmd", "b") {
		t.Errorf("expected binary RelDir cmd/b, got %s", binB.relDir)
	}
	msgsB := collectMessages(binB.expects)
	if !slices.Contains(msgsB, "b main assertion") {
		t.Errorf("expected %q in %v", "b main assertion", msgsB)
	}
	if !slices.Contains(msgsB, "shared assertion") {
		t.Errorf("expected %q in %v", "shared assertion", msgsB)
	}
	if len(binB.expects) != 2 {
		t.Errorf("binary B: expected 2 assertions, got %d: %v", len(binB.expects), msgsB)
	}

	// Cross-contamination check
	if slices.Contains(msgsB, "aonly assertion") {
		t.Errorf("did not expect %q in %v", "aonly assertion", msgsB)
	}
	if slices.Contains(msgsB, "a main assertion") {
		t.Errorf("did not expect %q in %v", "a main assertion", msgsB)
	}
	if slices.Contains(msgsA, "b main assertion") {
		t.Errorf("did not expect %q in %v", "b main assertion", msgsA)
	}
}

func TestAliasedImport(t *testing.T) {
	dir := absTestdata("aliased_import")
	scanner := NewAssertionScanner(false, dir, dir)
	if err := scanner.ScanAll(); err != nil {
		t.Fatalf("ScanAll failed: %v", err)
	}

	bins := scanner.binaries
	if len(bins) != 1 {
		t.Fatalf("expected 1 binary, got %d", len(bins))
	}

	bc := bins[0]
	if got := len(bc.expects); got != 2 {
		t.Errorf("expected 2 assertions, got %d", got)
	}

	msgs := collectMessages(bc.expects)
	if !slices.Contains(msgs, "aliased always") {
		t.Errorf("expected %q in %v", "aliased always", msgs)
	}
	if !slices.Contains(msgs, "aliased unreachable") {
		t.Errorf("expected %q in %v", "aliased unreachable", msgs)
	}
}

func TestNoMain(t *testing.T) {
	dir := absTestdata("no_main")
	scanner := NewAssertionScanner(false, dir, dir)
	if err := scanner.ScanAll(); err != nil {
		t.Fatalf("ScanAll failed: %v", err)
	}

	bins := scanner.binaries
	if len(bins) != 0 {
		t.Errorf("expected 0 binaries, got %d", len(bins))
	}
}

func TestNoAssertions(t *testing.T) {
	dir := absTestdata("no_assertions")
	scanner := NewAssertionScanner(false, dir, dir)
	if err := scanner.ScanAll(); err != nil {
		t.Fatalf("ScanAll failed: %v", err)
	}

	bins := scanner.binaries
	if len(bins) != 1 {
		t.Fatalf("expected 1 binary, got %d", len(bins))
	}

	bc := bins[0]
	if len(bc.expects) != 0 {
		t.Errorf("expected 0 assertions, got %d", len(bc.expects))
	}
	if len(bc.guidance) != 0 {
		t.Errorf("expected 0 guidance, got %d", len(bc.guidance))
	}

	// HasAssertionsDefined should be false
	if scanner.HasAssertionsDefined() {
		t.Error("HasAssertionsDefined should be false")
	}
}

// TestCatalogStability verifies that scanning a package which already
// contains a generated catalog produces the same results as the first
// scan. The catalog's assert.AssertRaw calls must not be picked up as
// additional assertions.
func TestCatalogStability(t *testing.T) {
	// Locate the repo root so we can set up a replace directive.
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	// The SDK's internal package uses cgo. Disable it so that the test
	// does not require a C compiler.
	t.Setenv("CGO_ENABLED", "0")

	// Copy the single_main fixture into a temp directory with its own module.
	tmpDir := t.TempDir()
	src, err := os.ReadFile(absTestdata("single_main/main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), src, 0644); err != nil {
		t.Fatal(err)
	}

	sdkModule := "github.com/antithesishq/antithesis-sdk-go"
	for _, cmd := range [][]string{
		{"go", "mod", "init", "example.com/stability-test"},
		{"go", "mod", "edit", "-require", sdkModule + "@v0.0.0"},
		{"go", "mod", "edit", "-replace", sdkModule + "=" + repoRoot},
		{"go", "mod", "tidy"},
		{"go", "mod", "vendor"},
	} {
		c := exec.Command(cmd[0], cmd[1:]...)
		c.Dir = tmpDir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("%s failed: %v\n%s", cmd, err, out)
		}
	}

	// First scan.
	scanner1 := NewAssertionScanner(false, tmpDir, tmpDir)
	if err := scanner1.ScanAll(); err != nil {
		t.Fatalf("first ScanAll failed: %v", err)
	}
	if len(scanner1.binaries) != 1 {
		t.Fatalf("expected 1 binary, got %d", len(scanner1.binaries))
	}
	firstMsgs := collectMessages(scanner1.binaries[0].expects)
	firstGuidanceCount := len(scanner1.binaries[0].guidance)

	// Write the catalog into the temp directory (same as source).
	scanner1.WriteAssertionCatalogs("stability test")

	// Verify the catalog file was created.
	catalogPath := filepath.Join(tmpDir, common.GENERATED_CATALOG_FILE)
	if _, err := os.Stat(catalogPath); err != nil {
		t.Fatalf("catalog not written: %v", err)
	}

	// Second scan — the catalog is now present alongside main.go.
	scanner2 := NewAssertionScanner(false, tmpDir, tmpDir)
	if err := scanner2.ScanAll(); err != nil {
		t.Fatalf("second ScanAll failed: %v", err)
	}
	if len(scanner2.binaries) != 1 {
		t.Fatalf("expected 1 binary on rescan, got %d", len(scanner2.binaries))
	}
	secondMsgs := collectMessages(scanner2.binaries[0].expects)
	secondGuidanceCount := len(scanner2.binaries[0].guidance)

	// The assertion counts must be identical.
	sort.Strings(firstMsgs)
	sort.Strings(secondMsgs)
	if !slices.Equal(firstMsgs, secondMsgs) {
		t.Errorf("assertion messages changed after catalog was written\nbefore: %v\nafter:  %v", firstMsgs, secondMsgs)
	}
	if firstGuidanceCount != secondGuidanceCount {
		t.Errorf("guidance count changed: before=%d, after=%d", firstGuidanceCount, secondGuidanceCount)
	}
}

func collectMessages(expects []*AntExpect) []string {
	msgs := make([]string, len(expects))
	for i, e := range expects {
		msgs[i] = e.Message
	}
	return msgs
}
