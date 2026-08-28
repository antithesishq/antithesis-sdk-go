package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var (
	sharedToolexec string
	sharedGoCache  string
	sharedRecorder string
)

func TestMain(m *testing.M) {
	os.Exit(func() int {
		work, err := os.MkdirTemp("", "antithesis-toolexec-test-")
		if err != nil {
			panic(err)
		}
		defer os.RemoveAll(work)

		sharedGoCache = filepath.Join(work, "gocache")
		sharedToolexec = filepath.Join(work, "antithesis-go-toolexec")
		build := exec.Command("go", "build", "-o", sharedToolexec, ".")
		build.Env = append(os.Environ(), "GOFLAGS=")
		if out, err := build.CombinedOutput(); err != nil {
			panic("building the wrapper failed: " + err.Error() + "\n" + string(out))
		}

		sharedRecorder = filepath.Join(work, "recorder")
		if err := writeRecorderModuleTo(sharedRecorder); err != nil {
			panic(err)
		}
		return m.Run()
	}())
}

// buildEnv is the environment every instrumented build in this package uses.
// GOFLAGS is cleared of any inherited -toolexec, which would recurse.
func buildEnv(symbolsDir, instrument string, extra ...string) []string {
	return append(append(os.Environ(),
		"GOFLAGS=-mod=mod",
		"GOCACHE="+sharedGoCache,
		"ANTITHESIS_INSTRUMENT="+instrument,
		"ANTITHESIS_SYMBOLS_DIR="+symbolsDir,
		"ANTITHESIS_SDK_MODULE_DIR="+sharedRecorder,
	), extra...)
}

// tablesForModule returns the symbol tables in symbolsDir that describe files
// under moduleDir, ignoring any left by another case sharing the build cache.
func tablesForModule(t *testing.T, symbolsDir, moduleDir string) []string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(symbolsDir, "*.sym.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	var mine []string
	for _, path := range paths {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(body), moduleDir+string(filepath.Separator)) {
			mine = append(mine, path)
		}
	}
	return mine
}
