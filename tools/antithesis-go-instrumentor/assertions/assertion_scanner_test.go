package assertions

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"

	qt "github.com/go-quicktest/qt"

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
	err := scanner.ScanAll()
	qt.Assert(t, qt.IsNil(err))

	bins := scanner.binaries
	qt.Assert(t, qt.HasLen(bins, 1))

	bc := bins[0]
	// single_main has 3 assertion calls: Always, Sometimes, Reachable
	qt.Check(t, qt.HasLen(bc.expects, 3))

	msgs := collectMessages(bc.expects)
	qt.Check(t, qt.SliceContains(msgs, "always true"))
	qt.Check(t, qt.SliceContains(msgs, "sometimes true"))
	qt.Check(t, qt.SliceContains(msgs, "reached main"))
}

func TestMultiMain(t *testing.T) {
	dir := absTestdata("multi_main")
	scanner := NewAssertionScanner(false, dir, dir)
	err := scanner.ScanAll()
	qt.Assert(t, qt.IsNil(err))

	bins := scanner.binaries
	qt.Assert(t, qt.HasLen(bins, 2))

	// Sort by RelDir for deterministic assertions
	sort.Slice(bins, func(i, j int) bool {
		return bins[i].relDir < bins[j].relDir
	})

	// cmd/a imports shared + aonly: 1 (a main) + 1 (shared) + 1 (aonly) = 3 assertions
	binA := bins[0]
	qt.Check(t, qt.Equals(binA.relDir, filepath.Join("cmd", "a")))
	msgsA := collectMessages(binA.expects)
	qt.Check(t, qt.SliceContains(msgsA, "a main assertion"))
	qt.Check(t, qt.SliceContains(msgsA, "shared assertion"))
	qt.Check(t, qt.SliceContains(msgsA, "aonly assertion"))
	qt.Check(t, qt.HasLen(binA.expects, 3))

	// cmd/b imports shared only: 1 (b main) + 1 (shared) = 2 assertions
	binB := bins[1]
	qt.Check(t, qt.Equals(binB.relDir, filepath.Join("cmd", "b")))
	msgsB := collectMessages(binB.expects)
	qt.Check(t, qt.SliceContains(msgsB, "b main assertion"))
	qt.Check(t, qt.SliceContains(msgsB, "shared assertion"))
	qt.Check(t, qt.HasLen(binB.expects, 2))

	// Cross-contamination check
	qt.Check(t, qt.Not(qt.SliceContains(msgsB, "aonly assertion")))
	qt.Check(t, qt.Not(qt.SliceContains(msgsB, "a main assertion")))
	qt.Check(t, qt.Not(qt.SliceContains(msgsA, "b main assertion")))
}

func TestAliasedImport(t *testing.T) {
	dir := absTestdata("aliased_import")
	scanner := NewAssertionScanner(false, dir, dir)
	err := scanner.ScanAll()
	qt.Assert(t, qt.IsNil(err))

	bins := scanner.binaries
	qt.Assert(t, qt.HasLen(bins, 1))

	bc := bins[0]
	qt.Check(t, qt.HasLen(bc.expects, 2))

	msgs := collectMessages(bc.expects)
	qt.Check(t, qt.SliceContains(msgs, "aliased always"))
	qt.Check(t, qt.SliceContains(msgs, "aliased unreachable"))
}

func TestNoMain(t *testing.T) {
	dir := absTestdata("no_main")
	scanner := NewAssertionScanner(false, dir, dir)
	err := scanner.ScanAll()
	qt.Assert(t, qt.IsNil(err))

	bins := scanner.binaries
	qt.Check(t, qt.HasLen(bins, 0))
}

func TestNoAssertions(t *testing.T) {
	dir := absTestdata("no_assertions")
	scanner := NewAssertionScanner(false, dir, dir)
	err := scanner.ScanAll()
	qt.Assert(t, qt.IsNil(err))

	bins := scanner.binaries
	qt.Assert(t, qt.HasLen(bins, 1))

	bc := bins[0]
	qt.Check(t, qt.HasLen(bc.expects, 0))
	qt.Check(t, qt.HasLen(bc.guidance, 0))

	// HasAssertionsDefined should be false
	qt.Check(t, qt.IsFalse(scanner.HasAssertionsDefined()))
}

// TestCatalogStability verifies that scanning a package which already
// contains a generated catalog produces the same results as the first
// scan. The catalog's assert.AssertRaw calls must not be picked up as
// additional assertions.
func TestCatalogStability(t *testing.T) {
	// Locate the repo root so we can set up a replace directive.
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	qt.Assert(t, qt.IsNil(err))

	// The SDK's internal package uses cgo. Disable it so that the test
	// does not require a C compiler.
	t.Setenv("CGO_ENABLED", "0")

	// Copy the single_main fixture into a temp directory with its own module.
	tmpDir := t.TempDir()
	src, err := os.ReadFile(absTestdata("single_main/main.go"))
	qt.Assert(t, qt.IsNil(err))
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), src, 0644)
	qt.Assert(t, qt.IsNil(err))

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
		out, err := c.CombinedOutput()
		qt.Assert(t, qt.IsNil(err), qt.Commentf("%s failed:\n%s", cmd, out))
	}

	// First scan.
	scanner1 := NewAssertionScanner(false, tmpDir, tmpDir)
	err = scanner1.ScanAll()
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.HasLen(scanner1.binaries, 1))
	firstMsgs := collectMessages(scanner1.binaries[0].expects)
	firstGuidanceCount := len(scanner1.binaries[0].guidance)

	// Write the catalog into the temp directory (same as source).
	scanner1.WriteAssertionCatalogs("stability test")

	// Verify the catalog file was created.
	catalogPath := filepath.Join(tmpDir, common.GENERATED_CATALOG_FILE)
	_, err = os.Stat(catalogPath)
	qt.Assert(t, qt.IsNil(err))

	// Second scan — the catalog is now present alongside main.go.
	scanner2 := NewAssertionScanner(false, tmpDir, tmpDir)
	err = scanner2.ScanAll()
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.HasLen(scanner2.binaries, 1))
	secondMsgs := collectMessages(scanner2.binaries[0].expects)
	secondGuidanceCount := len(scanner2.binaries[0].guidance)

	// The assertion counts must be identical.
	sort.Strings(firstMsgs)
	sort.Strings(secondMsgs)
	qt.Check(t, qt.DeepEquals(firstMsgs, secondMsgs))
	qt.Check(t, qt.Equals(firstGuidanceCount, secondGuidanceCount))
}

func collectMessages(expects []*AntExpect) []string {
	msgs := make([]string, len(expects))
	for i, e := range expects {
		msgs[i] = e.Message
	}
	return msgs
}
