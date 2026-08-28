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

	for _, be := range []struct {
		name       string
		args       []string
		mainGolden string
	}{
		{"ast_rewriting", nil, "customer/main.go"},
		{"source_editing", []string{"-source_editing"}, "customer/main.source_editing.go"},
	} {
		t.Run(be.name, func(t *testing.T) {
			// Copy input fixture to a temp dir so we don't modify testdata, and
			// rewrite the replace directive to use an absolute SDK path (the
			// relative path in the checked-in go.mod won't resolve from a temp dir).
			inputDir := filepath.Join(t.TempDir(), "input")
			copyDir(t, "testdata/input", inputDir)
			rewriteReplace(t, filepath.Join(inputDir, "go.mod"), sdkRoot)

			// Run the instrumentor with -local_sdk_path so it works in
			// sandboxed builds (e.g. Nix) where GOPROXY=off.
			outputDir := filepath.Join(t.TempDir(), "output")
			qt.Assert(t, qt.IsNil(os.MkdirAll(outputDir, 0755)))
			args := append(append([]string{}, be.args...), "-local_sdk_path", sdkRoot, inputDir, outputDir)
			runCmd(t, ".", instrumentorBin, args...)

			expectedDir := "testdata/expected_output"

			// Compare files that should match after normalization.
			for _, f := range []struct{ actual, golden string }{
				{"customer/main.go", be.mainGolden},
				{"customer/go.mod", "customer/go.mod"},
				{"notifier/notifier.go", "notifier/notifier.go"},
				{"notifier/go.mod", "notifier/go.mod"},
			} {
				t.Run(f.actual, func(t *testing.T) {
					expected, err := os.ReadFile(filepath.Join(expectedDir, f.golden))
					qt.Assert(t, qt.IsNil(err))
					actual, err := os.ReadFile(filepath.Join(outputDir, f.actual))
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
		})
	}
}

func TestE2EMultiModule(t *testing.T) {
	t.Setenv("CGO_ENABLED", "0")

	instrumentorBin := filepath.Join(t.TempDir(), "instrumentor")
	runCmd(t, ".", "go", "build", "-o", instrumentorBin, ".")

	sdkRoot, err := filepath.Abs(filepath.Join("..", ".."))
	qt.Assert(t, qt.IsNil(err))

	inputDir := filepath.Join(t.TempDir(), "input")
	copyDir(t, "testdata/input_multimodule", inputDir)
	rewriteReplace(t, filepath.Join(inputDir, "go.mod"), sdkRoot)
	rewriteReplace(t, filepath.Join(inputDir, "sub", "go.mod"), sdkRoot)

	outputDir := filepath.Join(t.TempDir(), "output")
	qt.Assert(t, qt.IsNil(os.MkdirAll(outputDir, 0755)))
	runCmd(t, ".", instrumentorBin, "-local_sdk_path", sdkRoot, inputDir, outputDir)

	t.Run("notifier_pinned_to_min", func(t *testing.T) {
		got := readGoDirective(t, filepath.Join(outputDir, "notifier", "go.mod"))
		qt.Check(t, qt.Equals(got, "1.24.0"))
		qt.Check(t, qt.Equals(readToolchain(t, filepath.Join(outputDir, "notifier", "go.mod")), ""))
	})

	// The customer go.mod files (pre-tidy) must already carry the input's
	// directives verbatim, since `go mod edit -print` does not modify them.
	t.Run("customer_directives_preserved_pretidy", func(t *testing.T) {
		qt.Check(t, qt.Equals(readGoDirective(t, filepath.Join(outputDir, "customer", "go.mod")), "1.24.7"))
		qt.Check(t, qt.Equals(readToolchain(t, filepath.Join(outputDir, "customer", "go.mod")), "go1.24.8"))
		qt.Check(t, qt.Equals(readGoDirective(t, filepath.Join(outputDir, "customer", "sub", "go.mod")), "1.24.3"))
	})

	// Strongest assertion: run `go mod tidy` on both customer modules after
	// instrumentation and verify zero drift.
	t.Run("post_tidy_zero_drift", func(t *testing.T) {
		runCmdEnv(t, filepath.Join(outputDir, "customer"), []string{"GOFLAGS="}, "go", "mod", "tidy")
		runCmdEnv(t, filepath.Join(outputDir, "customer", "sub"), []string{"GOFLAGS="}, "go", "mod", "tidy")

		qt.Check(t, qt.Equals(readGoDirective(t, filepath.Join(outputDir, "customer", "go.mod")), "1.24.7"))
		qt.Check(t, qt.Equals(readToolchain(t, filepath.Join(outputDir, "customer", "go.mod")), "go1.24.8"))
		qt.Check(t, qt.Equals(readGoDirective(t, filepath.Join(outputDir, "customer", "sub", "go.mod")), "1.24.3"))
	})

	// `go build ./...` on each customer module proves the notifier and the
	// instrumented customer code actually compile together — not just that
	// their go.mods parse correctly.
	t.Run("customer_builds", func(t *testing.T) {
		runCmdEnv(t, filepath.Join(outputDir, "customer"), []string{"GOFLAGS="}, "go", "build", "./...")
		runCmdEnv(t, filepath.Join(outputDir, "customer", "sub"), []string{"GOFLAGS="}, "go", "build", "./...")
	})
}

// readGoDirective returns the value of the `go` directive in a go.mod file,
// or "" if absent. The match is intentionally simple — full modfile parsing
// would require importing the package we're testing.
func readGoDirective(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	qt.Assert(t, qt.IsNil(err))
	re := regexp.MustCompile(`(?m)^go (\S+)\s*$`)
	m := re.FindStringSubmatch(string(data))
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

// readToolchain returns the value of the `toolchain` directive in a go.mod
// file, or "" if absent.
func readToolchain(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	qt.Assert(t, qt.IsNil(err))
	re := regexp.MustCompile(`(?m)^toolchain (\S+)\s*$`)
	m := re.FindStringSubmatch(string(data))
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

// TestE2EDeepSubmodule covers ENG-3940: the instrumentor's
// PathFromBaseDirectory previously used a `filepath.Match` pattern of
// `baseDir/*`, which does not cross path separators. For any customer
// submodule nested two or more levels deep, the function fell through
// to returning the *absolute* path of the submodule, which the caller
// then joined onto the customer output dir — producing
// `customer/<input-abspath>/...` rather than `customer/<rel-path>/...`.
//
// The fixture has root + `tools/inner/` (level 2). After instrumentation
// we assert: (a) the modified go.mod is at `customer/tools/inner/go.mod`,
// (b) it contains the notifier `require` and `replace` lines, and
// (c) no stray `customer/<abs-path>/tools/inner/` directory exists.
//
// This test does NOT run `go mod tidy` post-instrumentation — directive
// drift caused by ONCALL-1121 (notifier `go.mod` poisoning) is being
// fixed separately, and conflating the two would muddy this test's
// signal. It focuses purely on file placement.
func TestE2EDeepSubmodule(t *testing.T) {
	t.Setenv("CGO_ENABLED", "0")

	instrumentorBin := filepath.Join(t.TempDir(), "instrumentor")
	runCmd(t, ".", "go", "build", "-o", instrumentorBin, ".")

	sdkRoot, err := filepath.Abs(filepath.Join("..", ".."))
	qt.Assert(t, qt.IsNil(err))

	inputDir := filepath.Join(t.TempDir(), "input")
	copyDir(t, "testdata/input_deep_submodule", inputDir)
	rewriteReplace(t, filepath.Join(inputDir, "go.mod"), sdkRoot)
	rewriteReplace(t, filepath.Join(inputDir, "tools", "inner", "go.mod"), sdkRoot)

	outputDir := filepath.Join(t.TempDir(), "output")
	qt.Assert(t, qt.IsNil(os.MkdirAll(outputDir, 0755)))
	runCmd(t, ".", instrumentorBin, "-local_sdk_path", sdkRoot, inputDir, outputDir)

	// (a) The modified go.mod for the deep submodule must live at the
	// correct relative path under customer/.
	t.Run("deep_submodule_at_relative_path", func(t *testing.T) {
		correctPath := filepath.Join(outputDir, "customer", "tools", "inner", "go.mod")
		_, err := os.Stat(correctPath)
		qt.Check(t, qt.IsNil(err), qt.Commentf("expected modified go.mod at %s", correctPath))
	})

	// (b) That go.mod must contain the notifier require + replace —
	// proof that the instrumentor actually wrote it there (as opposed
	// to CopyRecursiveDir just dropping in the unmodified input copy).
	t.Run("deep_submodule_gomod_has_notifier", func(t *testing.T) {
		correctPath := filepath.Join(outputDir, "customer", "tools", "inner", "go.mod")
		data, err := os.ReadFile(correctPath)
		qt.Assert(t, qt.IsNil(err))
		s := string(data)
		qt.Check(t, qt.IsTrue(strings.Contains(s, "antithesis.notifier/")),
			qt.Commentf("go.mod missing antithesis.notifier require/replace; content:\n%s", s))
		qt.Check(t, qt.IsTrue(strings.Contains(s, "replace antithesis.notifier/")),
			qt.Commentf("go.mod missing antithesis.notifier replace; content:\n%s", s))
	})

	// (c) No `customer/<input-abspath>/...` mirror should exist. Walk
	// customer/ and flag any directory whose path starts with what
	// looks like an absolute input path (i.e. anything beginning with
	// `customer/` + an absolute filesystem prefix).
	t.Run("no_absolute_path_mirror", func(t *testing.T) {
		customerDir := filepath.Join(outputDir, "customer")

		// The pre-fix bug produced paths like
		// `customer/tmp/.../input/tools/inner/`. Detect by looking for
		// any directory under customer/ that contains an "input"
		// segment (matching how t.TempDir names directories) — or more
		// strictly, any directory whose path component starts with
		// what would be an absolute root.
		var offenders []string
		_ = filepath.WalkDir(customerDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || !d.IsDir() {
				return nil
			}
			// The bug's signature: under `customer/` we'd see the
			// instrumented input's absolute path replicated, e.g.
			// `customer/tmp/...`. Absolute-path mirrors start with
			// what was the first segment of the input abs path.
			rel, _ := filepath.Rel(customerDir, path)
			if rel == "." {
				return nil
			}
			// Anything matching the leading segment of the temp dir
			// (which contains the input fixture) is the bug.
			parts := strings.Split(rel, string(filepath.Separator))
			if len(parts) > 0 && (parts[0] == "tmp" || parts[0] == "home" || parts[0] == "build" || parts[0] == "private") {
				offenders = append(offenders, rel)
			}
			return nil
		})
		qt.Check(t, qt.HasLen(offenders, 0),
			qt.Commentf("path-doubling bug: found customer/<abs-path>/... directories: %v", offenders))
	})
}

// runCmd runs a command and fails the test if it exits non-zero.
func runCmd(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	runCmdEnv(t, dir, nil, name, args...)
}

// runCmdEnv runs a command with additional env vars (formatted as "KEY=VAL";
// empty value clears the var). Fails the test if the command exits non-zero.
func runCmdEnv(t *testing.T, dir string, extraEnv []string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), extraEnv...)
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

// Normalization regexes
var (
	notifierHashRe = regexp.MustCompile(`z[0-9a-f]{12}\b`)
	symbolHashRe   = regexp.MustCompile(`go-[0-9a-f]{12}\b`)
	sdkVersionRe   = regexp.MustCompile(`antithesis-sdk-go v[\d.]+`)
	absPathRe      = regexp.MustCompile(`/[^\t\n]*/input/`)
	instrumentorRe = regexp.MustCompile(`# instrumentor = .+`)
	sdkReplaceRe   = regexp.MustCompile(`(?m)^replace github\.com/antithesishq/antithesis-sdk-go => .+$`)
)

func normalizeContent(s string) string {
	s = notifierHashRe.ReplaceAllString(s, "zHASH")
	s = symbolHashRe.ReplaceAllString(s, "go-HASH")
	s = sdkVersionRe.ReplaceAllString(s, "antithesis-sdk-go vX.Y.Z")
	s = absPathRe.ReplaceAllString(s, "INPUT_DIR/")
	s = instrumentorRe.ReplaceAllString(s, "# instrumentor = INSTRUMENTOR")
	s = sdkReplaceRe.ReplaceAllString(s, "replace github.com/antithesishq/antithesis-sdk-go => SDK_ROOT")
	return s
}
