package main

// Multi-package end-to-end test.
//
// The runtime_edges fixtures are all a single `package main`, so they exercise
// the edge finder but cannot show what is structurally new about the -toolexec
// front end: one coverage module and one symbol table *per package*, each
// numbering its edges from 1, with a shared dependency appearing once rather than
// once per dependent.
//
// testdata/multipackage is a four-package diamond -- main depends on alpha and
// beta, both of which depend on shared -- and its golden artifacts are therefore
// the only checked-in ones that reflect that structure.
//
// Regenerate after changing the fixture:
//
//	go test ./tools/antithesis-go-toolexec -run TestMultiPackage -update

import (
	"encoding/json"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	qt "github.com/go-quicktest/qt"
)

var update = flag.Bool("update", false, "rewrite the multipackage golden artifacts")

const multiPackageDir = "testdata/multipackage"

// multiPackageModule is the import path the fixture's packages use.
const multiPackageModule = "multipkg"

// multiPackageCase is one argv invocation and the notifications it must produce,
// each written "<package> <edge>" so that per-package numbering is legible and
// stable against symbol table renaming.
type multiPackageCase struct {
	Args  []string `json:"args"`
	Fired []string `json:"fired"`
}

type multiPackageCases struct {
	// Modules maps each package to the number of edges its module registers.
	Modules map[string]int     `json:"modules"`
	Cases   []multiPackageCase `json:"cases"`
}

func TestMultiPackage(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a wrapper binary and a multi-package program")
	}

	binary, symbolsDir := buildMultiPackage(t)

	// One symbol table per package, named by content hash. Map each to the
	// package it describes, using the file column of its rows -- which is stable
	// where the hash in the file name is not.
	// Symbol table rows record absolute build paths, as the whole-tree
	// instrumentor's do, so they are rewritten relative to the module root before
	// anything is compared or checked in.
	tables := loadSymbolTables(t, symbolsDir, filepath.Dir(binary))
	qt.Assert(t, qt.HasLen(tables, 4),
		qt.Commentf("expected one symbol table per package, got %d", len(tables)))

	names := symbolTableNames(t, symbolsDir, filepath.Dir(binary))

	got := multiPackageCases{Modules: map[string]int{}}
	for pkg, table := range tables {
		got.Modules[pkg] = len(table.rows)
	}

	wanted := loadMultiPackageCases(t)
	for _, c := range wanted.Cases {
		got.Cases = append(got.Cases, multiPackageCase{
			Args:  c.Args,
			Fired: runForModuleEdges(t, binary, c.Args, names),
		})
	}

	if *update {
		writeMultiPackageCases(t, got)
		for pkg, table := range tables {
			writeGolden(t, filepath.Join(multiPackageDir, "expected", pkg+".sym.tsv"), table.text)
		}
		t.Log("golden artifacts rewritten")
		return
	}

	// Every package registers its own module, sized to its own edge count.
	qt.Check(t, qt.DeepEquals(got.Modules, wanted.Modules))

	// Each package's edges are numbered from 1, independently.
	for pkg, table := range tables {
		qt.Check(t, qt.DeepEquals(table.edgeIDs(), sequence(len(table.rows))),
			qt.Commentf("package %s must number its edges 1..N", pkg))
	}

	for i, c := range wanted.Cases {
		qt.Check(t, qt.DeepEquals(got.Cases[i].Fired, c.Fired), qt.Commentf("argv %q", c.Args))
	}

	// And each symbol table matches its checked-in artifact.
	for pkg, table := range tables {
		golden := filepath.Join(multiPackageDir, "expected", pkg+".sym.tsv")
		want, err := os.ReadFile(golden)
		qt.Assert(t, qt.IsNil(err), qt.Commentf("missing golden %s; run with -update", golden))
		qt.Check(t, qt.Equals(table.text, string(want)), qt.Commentf("package %s", pkg))
	}
}

// symbolTable is one package's table: its normalized text, plus its rows.
type symbolTable struct {
	text string
	rows []string
}

func (s symbolTable) edgeIDs() []int {
	ids := make([]int, 0, len(s.rows))
	for _, row := range s.rows {
		columns := strings.Split(row, "\t")
		id, err := strconv.Atoi(columns[len(columns)-1])
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

// loadSymbolTables reads every table in dir and keys them by the package they
// describe, taken from the directory component of their first row's file.
func loadSymbolTables(t *testing.T, dir, moduleRoot string) map[string]symbolTable {
	t.Helper()
	paths := tablesForModule(t, dir, moduleRoot)

	tables := make(map[string]symbolTable, len(paths))
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		qt.Assert(t, qt.IsNil(err))
		body := relativizePaths(string(raw), moduleRoot)
		rows := symbolTableRows(body)
		qt.Assert(t, qt.IsTrue(len(rows) > 0), qt.Commentf("%s has no rows", path))

		pkg := packageOfRow(t, body)
		_, duplicate := tables[pkg]
		qt.Assert(t, qt.IsFalse(duplicate),
			qt.Commentf("two symbol tables describe package %s; a shared dependency must appear once", pkg))
		tables[pkg] = symbolTable{text: normalizeSymbolTable(body, pkg), rows: rows}
	}
	return tables
}

// packageOfRow names the package a table describes. Rows carry module-relative
// file paths, so "shared/shared.go" is package shared and "main.go" is main.
func packageOfRow(t *testing.T, table string) string {
	t.Helper()
	for _, line := range strings.Split(table, "\n") {
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "file\t") {
			continue
		}
		file := strings.SplitN(line, "\t", 2)[0]
		if dir := filepath.Dir(file); dir != "." {
			return dir
		}
		return "main"
	}
	t.Fatalf("no data rows in symbol table")
	return ""
}

// relativizePaths rewrites the absolute build paths a symbol table records so
// they are relative to the module root, which is what makes the table stable
// enough to check in.
func relativizePaths(table, moduleRoot string) string {
	return strings.ReplaceAll(table, moduleRoot+string(filepath.Separator), "")
}

// normalizeSymbolTable makes a table comparable to a checked-in golden: the
// content-hash module name and the absolute instrumentor path both vary by
// build, so neither can be asserted verbatim.
func normalizeSymbolTable(table, pkg string) string {
	var out []string
	for _, line := range strings.Split(table, "\n") {
		switch {
		case strings.HasPrefix(line, "# module = "):
			line = "# module = " + pkg
		case strings.HasPrefix(line, "# instrumentor = "):
			line = "# instrumentor = INSTRUMENTOR"
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// runForModuleEdges runs the program and returns its notifications as
// "<package> <edge>", in order.
func runForModuleEdges(t *testing.T, binary string, args []string, names map[string]string) []string {
	t.Helper()
	command := exec.Command(binary, args...)
	command.Dir = filepath.Dir(binary)
	out, err := command.Output()
	qt.Assert(t, qt.IsNil(err), qt.Commentf("running %q failed", args))

	// The recorder reports the symbol table name; translate it to the package it
	// describes so the expectation survives a rehash.
	fired := []string{}
	for _, n := range recorded(t, out) {
		pkg, ok := names[n.symbolTable]
		qt.Assert(t, qt.IsTrue(ok), qt.Commentf("unknown symbol table %q", n.symbolTable))
		fired = append(fired, pkg+" "+strconv.Itoa(n.edge))
	}
	return fired
}

// symbolTableNames maps each symbol table's file name to its package.
func symbolTableNames(t *testing.T, dir, moduleRoot string) map[string]string {
	t.Helper()
	paths := tablesForModule(t, dir, moduleRoot)
	names := make(map[string]string, len(paths))
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		qt.Assert(t, qt.IsNil(err))
		names[filepath.Base(path)] = packageOfRow(t, relativizePaths(string(raw), moduleRoot))
	}
	return names
}

// buildMultiPackage assembles the fixture into a temporary module, builds it
// through the wrapper, and returns the binary and its symbols directory.
func buildMultiPackage(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()

	goMod := "module " + multiPackageModule + "\n\ngo 1.24\n\n" +
		"require github.com/antithesishq/antithesis-sdk-go v0.0.0\n\n" +
		"replace github.com/antithesishq/antithesis-sdk-go => " + sharedRecorder + "\n"
	qt.Assert(t, qt.IsNil(os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644)))
	copyTree(t, multiPackageDir, dir)

	symbolsDir := filepath.Join(dir, "symbols")
	binary := filepath.Join(dir, "prog")
	build := exec.Command("go", "build", "-toolexec="+sharedToolexec, "-o", binary, ".")
	build.Dir = dir
	build.Env = buildEnv(symbolsDir, multiPackageModule, "ANTITHESIS_SKIP_CATALOG=1")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building the instrumented program failed: %v\n%s", err, out)
	}
	return binary, symbolsDir
}

// copyTree copies the fixture's Go sources, preserving directory structure. Only
// .go files are copied: cases.json and expected/ are the test's own data.
func copyTree(t *testing.T, from, to string) {
	t.Helper()
	err := filepath.WalkDir(from, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		rel, err := filepath.Rel(from, path)
		if err != nil {
			return err
		}
		target := filepath.Join(to, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, body, 0o644)
	})
	qt.Assert(t, qt.IsNil(err))
}

func loadMultiPackageCases(t *testing.T) multiPackageCases {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(multiPackageDir, "cases.json"))
	qt.Assert(t, qt.IsNil(err), qt.Commentf("missing cases.json; run with -update"))
	var cases multiPackageCases
	qt.Assert(t, qt.IsNil(json.Unmarshal(body, &cases)))
	qt.Assert(t, qt.IsTrue(len(cases.Cases) > 0))
	return cases
}

func writeMultiPackageCases(t *testing.T, cases multiPackageCases) {
	t.Helper()
	body, err := json.MarshalIndent(cases, "", "  ")
	qt.Assert(t, qt.IsNil(err))
	writeGolden(t, filepath.Join(multiPackageDir, "cases.json"), string(body)+"\n")
}

func writeGolden(t *testing.T, path, content string) {
	t.Helper()
	qt.Assert(t, qt.IsNil(os.MkdirAll(filepath.Dir(path), 0o755)))
	qt.Assert(t, qt.IsNil(os.WriteFile(path, []byte(content), 0o644)))
}

// sequence returns 1..n, the edge ids a package's module must use.
func sequence(n int) []int {
	ids := make([]int, 0, n)
	for i := 1; i <= n; i++ {
		ids = append(ids, i)
	}
	return ids
}
