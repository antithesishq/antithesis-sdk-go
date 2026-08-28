package instrumentor_test

// Runtime edge-notification tests.
//
// These check that an instrumented program *fires the expected edges at runtime*. Each case under
// testdata/runtime_edges/ is a self-contained, side-effect-free, argv-driven program:
//
//	testdata/runtime_edges/<case>/program.go        -- runnable `package main`
//	testdata/runtime_edges/<case>/<embed assets>    -- files referenced via //go:embed
//	testdata/runtime_edges/<case>/cases.json        -- argv -> expected edge ids (in order)
//	testdata/runtime_edges/<case>/expected.sym.tsv  -- the symbol table (edge id -> source span)
//
// Regenerate after changing a program or (intentionally) changing instrumentor
// runtime behavior:
//
//	go test ./scanners/coverage/instrumentor -run TestRuntimeEdges -update
//
// (-update rewrites each cases.json's edge lists and expected.sym.tsv.)

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/scanners/coverage/symboltable"

	qt "github.com/go-quicktest/qt"
)

// recorderSrc is the stand-in for libvoidstar: the instrumented program's
// Notify(edge) calls land here, and each edge id is streamed to stdout.
const recorderSrc = `package rec

import (
	"fmt"
	"os"
)

func Notify(edge int) { fmt.Fprintln(os.Stdout, edge) }
`

type edgeCase struct {
	Args  []string `json:"args"`
	Edges []int    `json:"edges"`
}

type edgeCases struct {
	Cases []edgeCase `json:"cases"`
}

func TestRuntimeEdges(t *testing.T) {
	programs, err := filepath.Glob("testdata/runtime_edges/*/program.go")
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.IsTrue(len(programs) > 0))

	for _, progPath := range programs {
		caseDir := filepath.Dir(progPath)
		t.Run(filepath.Base(caseDir), func(t *testing.T) {
			cases := loadCases(t, filepath.Join(caseDir, "cases.json"))

			syms := make([]string, len(backends))
			edges := make([][][]int, len(backends))
			for bi, be := range backends {
				syms[bi], edges[bi] = instrumentAndRun(t, caseDir, progPath, be.name, be.create, cases)
			}

			checkGolden(t, filepath.Join(caseDir, "expected.sym.tsv"), syms[0])
			qt.Check(t, qt.Equals(syms[1], syms[0]),
				qt.Commentf("%s and %s must emit identical symbol tables", backends[0].name, backends[1].name))

			for i := range cases.Cases {
				if *update {
					cases.Cases[i].Edges = edges[0][i]
				} else {
					qt.Check(t, qt.DeepEquals(edges[0][i], cases.Cases[i].Edges),
						qt.Commentf("%s argv %v", backends[0].name, cases.Cases[i].Args))
				}
				qt.Check(t, qt.DeepEquals(edges[1][i], edges[0][i]),
					qt.Commentf("%s vs %s argv %v", backends[1].name, backends[0].name, cases.Cases[i].Args))
			}
			if *update {
				writeCases(t, filepath.Join(caseDir, "cases.json"), cases)
			}
		})
	}
}

func instrumentAndRun(t *testing.T, caseDir, progPath, name string, create func(basePath, shimPkg string, table *symboltable.SymbolTable) instrumenter, cases edgeCases) (string, [][]int) {
	t.Helper()

	// Build a self-contained temp module: recorder + the instrumented program +
	// any embed assets.
	dir := t.TempDir()
	write(t, filepath.Join(dir, "go.mod"), "module edgetest\n\ngo 1.24\n")
	qt.Assert(t, qt.IsNil(os.MkdirAll(filepath.Join(dir, "rec"), 0755)))
	write(t, filepath.Join(dir, "rec", "rec.go"), recorderSrc)
	copyEmbedAssets(t, caseDir, dir)

	// Instrument program.go with the notifier shim pointed at the recorder.
	progTmp := filepath.Join(dir, "program.go")
	write(t, progTmp, read(t, progPath))
	symPath := filepath.Join(dir, "edges.sym.tsv")
	sym, err := symboltable.CreateSymbolTableFile(symPath, "testmodule")
	qt.Assert(t, qt.IsNil(err))
	out, err := create(dir, "edgetest/rec", sym).Instrument(progTmp)
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.IsTrue(out != ""), qt.Commentf("%s: program.go was skipped (not instrumented)", name))
	write(t, progTmp, out)
	qt.Assert(t, qt.IsNil(sym.Close()))

	symNorm := strings.ReplaceAll(read(t, symPath), progTmp, "program.go")
	symNorm = instrumentorLineRe.ReplaceAllString(symNorm, "# instrumentor = INSTRUMENTOR")

	// Build once, then run every argv case against the recorder.
	bin := filepath.Join(dir, "prog")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = dir
	build.Env = append(os.Environ(), "GOFLAGS=")
	if outB, err := build.CombinedOutput(); err != nil {
		t.Fatalf("%s: building instrumented program failed: %v\n%s", name, err, outB)
	}

	got := make([][]int, len(cases.Cases))
	for i := range cases.Cases {
		got[i] = runEdges(t, bin, cases.Cases[i].Args)
	}
	return symNorm, got
}

// runEdges runs the built program with args and returns the edge ids it printed.
func runEdges(t *testing.T, bin string, args []string) []int {
	t.Helper()
	cmd := exec.Command(bin, args...)
	out, err := cmd.Output()
	if err != nil {
		stderr := ""
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = string(ee.Stderr)
		}
		t.Fatalf("running %s %v failed: %v\n%s", filepath.Base(bin), args, err, stderr)
	}
	var edges []int
	for _, line := range strings.Fields(strings.TrimSpace(string(out))) {
		n, err := strconv.Atoi(line)
		qt.Assert(t, qt.IsNil(err), qt.Commentf("non-integer edge output: %q", line))
		edges = append(edges, n)
	}
	return edges
}

func copyEmbedAssets(t *testing.T, caseDir, dir string) {
	t.Helper()
	entries, err := os.ReadDir(caseDir)
	qt.Assert(t, qt.IsNil(err))
	for _, e := range entries {
		switch {
		case e.IsDir(), e.Name() == "program.go", e.Name() == "cases.json", e.Name() == "expected.sym.tsv":
			continue
		}
		write(t, filepath.Join(dir, e.Name()), read(t, filepath.Join(caseDir, e.Name())))
	}
}

func loadCases(t *testing.T, path string) edgeCases {
	t.Helper()
	var c edgeCases
	qt.Assert(t, qt.IsNil(json.Unmarshal([]byte(read(t, path)), &c)))
	return c
}

func writeCases(t *testing.T, path string, c edgeCases) {
	t.Helper()
	content, _ := json.MarshalIndent(c, "", "  ")
	write(t, path, string(content))
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	qt.Assert(t, qt.IsNil(err))
	return string(b)
}

func write(t *testing.T, path, content string) {
	t.Helper()
	qt.Assert(t, qt.IsNil(os.WriteFile(path, []byte(content), 0644)))
}
