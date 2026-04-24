package main

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	qt "github.com/go-quicktest/qt"
)

func TestE2E(t *testing.T) {
	t.Setenv("CGO_ENABLED", "0")

	// Build the instrumentor binary.
	instrumentorBin := filepath.Join(t.TempDir(), "instrumentor")
	runCmd(t, ".", "go", "build", "-o", instrumentorBin, ".")

	// Absolute path to the SDK repo root (sdk/go/repo).
	sdkRoot, err := filepath.Abs(filepath.Join("..", ".."))
	qt.Assert(t, qt.IsNil(err))

	// Copy input fixture to a temp dir so we don't modify testdata, and
	// rewrite the replace directive to use an absolute SDK path (the
	// relative path in the checked-in go.mod won't resolve from a temp dir).
	inputDir := filepath.Join(t.TempDir(), "input")
	copyDir(t, "testdata/input", inputDir)
	rewriteReplace(t, filepath.Join(inputDir, "go.mod"), sdkRoot)

	// Run the instrumentor with -local_sdk_path so it works in
	// sandboxed builds (e.g. Nix) where GOPROXY=off.
	outputDir := filepath.Join(t.TempDir(), "output")
	err = os.MkdirAll(outputDir, 0755)
	qt.Assert(t, qt.IsNil(err))
	runCmd(t, ".", instrumentorBin, "-local_sdk_path", sdkRoot, inputDir, outputDir)

	expectedDir := "testdata/expected_output"

	// Compare files that should match after normalization.
	for _, relPath := range []string{
		"customer/main.go",
		"customer/go.mod",
		"notifier/notifier.go",
		"notifier/go.mod",
	} {
		t.Run(relPath, func(t *testing.T) {
			expected, err := os.ReadFile(filepath.Join(expectedDir, relPath))
			qt.Assert(t, qt.IsNil(err))
			actual, err := os.ReadFile(filepath.Join(outputDir, relPath))
			qt.Assert(t, qt.IsNil(err))
			qt.Check(t, qt.Equals(
				normalizeContent(string(actual)),
				normalizeContent(string(expected)),
			))
		})
	}

	// notifier/go.sum won't exist because we use the local sdk

	// Compare symbol table (filename contains a content hash, so glob for it).
	t.Run("symbols", func(t *testing.T) {
		expectedFiles, err := filepath.Glob(filepath.Join(expectedDir, "symbols", "*.sym.tsv"))
		qt.Assert(t, qt.IsNil(err))
		qt.Assert(t, qt.HasLen(expectedFiles, 1))

		actualFiles, err := filepath.Glob(filepath.Join(outputDir, "symbols", "*.sym.tsv"))
		qt.Assert(t, qt.IsNil(err))
		qt.Assert(t, qt.HasLen(actualFiles, 1))

		expected, err := os.ReadFile(expectedFiles[0])
		qt.Assert(t, qt.IsNil(err))
		actual, err := os.ReadFile(actualFiles[0])
		qt.Assert(t, qt.IsNil(err))

		qt.Check(t, qt.Equals(
			normalizeContent(string(actual)),
			normalizeContent(string(expected)),
		))
	})
}

// runCmd runs a command and fails the test if it exits non-zero.
func runCmd(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	qt.Assert(t, qt.IsNil(err), qt.Commentf("%s %s failed:\n%s", name, strings.Join(args, " "), out))
}

// copyDir recursively copies src to dst.
func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
	qt.Assert(t, qt.IsNil(err))
}

// rewriteReplace rewrites the replace directive for the SDK to use an
// absolute path, so that go mod tidy works from any output location.
func rewriteReplace(t *testing.T, gomodPath, sdkRoot string) {
	t.Helper()
	data, err := os.ReadFile(gomodPath)
	qt.Assert(t, qt.IsNil(err))

	re := regexp.MustCompile(`(?m)^replace github\.com/antithesishq/antithesis-sdk-go => .+$`)
	updated := re.ReplaceAllString(string(data),
		"replace github.com/antithesishq/antithesis-sdk-go => "+sdkRoot)
	if string(data) == updated {
		t.Fatalf("replace directive not found in %s", gomodPath)
	}
	qt.Assert(t, qt.IsNil(os.WriteFile(gomodPath, []byte(updated), 0644)))
}

// Normalization regexes.
var (
	notifierHashRe = regexp.MustCompile(`z[0-9a-f]{12}\b`)
	symbolHashRe   = regexp.MustCompile(`go-[0-9a-f]{12}\b`)
	sdkVersionRe   = regexp.MustCompile(`antithesis-sdk-go v[\d.]+`)
	goVersionRe    = regexp.MustCompile(`(?m)^go \d+\.\d+(?:\.\d+)?$`)
	absPathRe      = regexp.MustCompile(`/[^\t\n]*/input/`)
	instrumentorRe = regexp.MustCompile(`# instrumentor = .+`)
	sdkReplaceRe   = regexp.MustCompile(`(?m)^replace github\.com/antithesishq/antithesis-sdk-go => .+$`)
)

func normalizeContent(s string) string {
	s = notifierHashRe.ReplaceAllString(s, "zHASH")
	s = symbolHashRe.ReplaceAllString(s, "go-HASH")
	s = sdkVersionRe.ReplaceAllString(s, "antithesis-sdk-go vX.Y.Z")
	s = goVersionRe.ReplaceAllString(s, "go X.Y.Z")
	s = absPathRe.ReplaceAllString(s, "INPUT_DIR/")
	s = instrumentorRe.ReplaceAllString(s, "# instrumentor = INSTRUMENTOR")
	s = sdkReplaceRe.ReplaceAllString(s, "replace github.com/antithesishq/antithesis-sdk-go => SDK_ROOT")
	return s
}
