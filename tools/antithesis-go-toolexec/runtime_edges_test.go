package main

// End-to-end runtime edge tests for the -toolexec front end.
//
// These reuse the fixtures TestRuntimeEdges already curates for the whole-tree
// instrumentor and drive them through a real `go build -toolexec=antithesis-go-toolexec`
// instead. Regenerating the fixtures with
//
//	go test ./scanners/coverage/instrumentor -run TestRuntimeEdges -update
//
// updates the expectations for both front ends at once.

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	qt "github.com/go-quicktest/qt"
)

// runtimeEdgesDir holds the fixtures, which live with the instrumentor whose
// behavior they describe.
const runtimeEdgesDir = "../antithesis-go-instrumentor/scanners/coverage/instrumentor/testdata/runtime_edges"

// The recorder module shadows the SDK import path that instrumented code calls
// into, so that every Notify lands somewhere observable. The real SDK cannot be
// used here: without a loaded library its notifications go nowhere, and with one
// they would be absorbed by coverage leases rather than reported individually.
const (
	recorderGoMod = `module github.com/antithesishq/antithesis-sdk-go

go 1.24
`

	recorderInstrumentation = `package instrumentation

import (
	"fmt"
	"os"
)

// Module mirrors the shape the generated registration file expects. Notify
// prints the module's symbol table and the module-local edge id, in call order,
// with nothing suppressed. The symbol table is needed because edge ids restart
// at 1 in every package, so an id alone does not identify an edge.
type Module struct{ symbolTable string }

func RegisterModule(symbolTable string, edgeCount int) *Module {
	return &Module{symbolTable: symbolTable}
}

func (m *Module) Notify(edge int) {
	fmt.Fprintf(os.Stdout, "%s\t%d\n", m.symbolTable, edge)
}
`
)

type toolexecEdgeCase struct {
	Args  []string `json:"args"`
	Edges []int    `json:"edges"`
}

type toolexecEdgeCases struct {
	Cases []toolexecEdgeCase `json:"cases"`
}

func TestRuntimeEdgesToolexec(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a wrapper binary and one program per case")
	}

	programs, err := filepath.Glob(filepath.Join(runtimeEdgesDir, "*/program.go"))
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.IsTrue(len(programs) > 0))

	for _, program := range programs {
		caseDir := filepath.Dir(program)
		t.Run(filepath.Base(caseDir), func(t *testing.T) {
			t.Parallel()

			var cases toolexecEdgeCases
			body, err := os.ReadFile(filepath.Join(caseDir, "cases.json"))
			qt.Assert(t, qt.IsNil(err))
			qt.Assert(t, qt.IsNil(json.Unmarshal(body, &cases)))

			binary, symbolRows := buildInstrumented(t, caseDir, program)

			// The same edges must fire, in the same order, as the whole-tree
			// instrumentor produces for this program.
			for _, c := range cases.Cases {
				qt.Check(t, qt.DeepEquals(runForEdges(t, binary, c.Args), c.Edges),
					qt.Commentf("argv %q", c.Args))
			}

			// And they must describe the same source spans. Only the rows are
			// compared: the header names the module, which is per-package here
			// and per-program there.
			want, err := os.ReadFile(filepath.Join(caseDir, "expected.sym.tsv"))
			qt.Assert(t, qt.IsNil(err))
			qt.Check(t, qt.DeepEquals(symbolRows, symbolTableRows(string(want))))
		})
	}
}

// writeRecorderModuleTo materializes the module that shadows the SDK.
func writeRecorderModuleTo(dir string) error {
	if err := os.MkdirAll(filepath.Join(dir, "instrumentation"), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(recorderGoMod), 0o644); err != nil {
		return err
	}
	return os.WriteFile(
		filepath.Join(dir, "instrumentation", "instrumentation.go"), []byte(recorderInstrumentation), 0o644)
}

// buildInstrumented assembles a temporary module around one fixture program,
// builds it through the wrapper, and returns the binary along with the rows of
// the symbol table that was produced.
func buildInstrumented(t *testing.T, caseDir, program string) (string, []string) {
	t.Helper()
	dir := t.TempDir()

	// The program does not import the SDK -- instrumentation is what introduces
	// that dependency -- so the requirement is stated here and satisfied by the
	// recorder. Left untidied on purpose: `go mod tidy` would drop it.
	goMod := "module edgetest\n\ngo 1.24\n\nrequire github.com/antithesishq/antithesis-sdk-go v0.0.0\n\n" +
		"replace github.com/antithesishq/antithesis-sdk-go => " + sharedRecorder + "\n"
	qt.Assert(t, qt.IsNil(os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644)))

	source, err := os.ReadFile(program)
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.IsNil(os.WriteFile(filepath.Join(dir, "program.go"), source, 0o644)))
	copyEmbedAssets(t, caseDir, dir)

	symbolsDir := filepath.Join(dir, "symbols")
	binary := filepath.Join(dir, "prog")
	build := exec.Command("go", "build", "-toolexec="+sharedToolexec, "-o", binary, ".")
	build.Dir = dir
	// These programs carry no assertions, and the recorder deliberately does not
	// provide the assert package.
	build.Env = buildEnv(symbolsDir, "edgetest", "ANTITHESIS_SKIP_CATALOG=1")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building the instrumented program failed: %v\n%s", err, out)
	}

	tables := tablesForModule(t, symbolsDir, dir)
	qt.Assert(t, qt.HasLen(tables, 1))
	body, err := os.ReadFile(tables[0])
	qt.Assert(t, qt.IsNil(err))
	return binary, symbolTableRows(string(body))
}

// runForEdges runs the instrumented program and returns the edge ids it printed.
func runForEdges(t *testing.T, binary string, args []string) []int {
	t.Helper()
	command := exec.Command(binary, args...)
	command.Dir = filepath.Dir(binary)
	out, err := command.Output()
	if err != nil {
		stderr := ""
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderr = string(exitErr.Stderr)
		}
		t.Fatalf("running %s %q failed: %v\n%s", filepath.Base(binary), args, err, stderr)
	}
	// Empty rather than nil: a fixture expecting no edges unmarshals to an empty
	// slice, and DeepEquals distinguishes the two.
	edges := []int{}
	for _, notification := range recorded(t, out) {
		edges = append(edges, notification.edge)
	}
	return edges
}

// notification is one line of recorder output: which module fired, and which of
// its edges.
type notification struct {
	symbolTable string
	edge        int
}

func recorded(t *testing.T, out []byte) []notification {
	t.Helper()
	var notifications []notification
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		table, id, found := strings.Cut(line, "\t")
		qt.Assert(t, qt.IsTrue(found), qt.Commentf("malformed recorder line %q", line))
		edge, err := strconv.Atoi(id)
		qt.Assert(t, qt.IsNil(err), qt.Commentf("non-integer edge %q", id))
		notifications = append(notifications, notification{symbolTable: table, edge: edge})
	}
	return notifications
}

// symbolTableRows returns a symbol table's data rows with the file column reduced
// to a base name, so tables written from different directories compare equal.
func symbolTableRows(table string) []string {
	var rows []string
	for _, line := range strings.Split(table, "\n") {
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "file\t") {
			continue
		}
		columns := strings.Split(line, "\t")
		columns[0] = filepath.Base(columns[0])
		rows = append(rows, strings.Join(columns, "\t"))
	}
	return rows
}

func copyEmbedAssets(t *testing.T, caseDir, dir string) {
	t.Helper()
	entries, err := os.ReadDir(caseDir)
	qt.Assert(t, qt.IsNil(err))
	for _, entry := range entries {
		switch {
		case entry.IsDir(), entry.Name() == "program.go",
			entry.Name() == "cases.json", entry.Name() == "expected.sym.tsv":
			continue
		}
		body, err := os.ReadFile(filepath.Join(caseDir, entry.Name()))
		qt.Assert(t, qt.IsNil(err))
		qt.Assert(t, qt.IsNil(os.WriteFile(filepath.Join(dir, entry.Name()), body, 0o644)))
	}
}
