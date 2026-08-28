package instrumentor_test

// Golden-file tests for the coverage instrumentor.
//
// Both coverage back-ends -- the AST-rewriting instrumentor and the
// source-editing instrumentor -- are exercised against the same inputs. The
// instrumented *source* they emit differs (the AST path reprints and adds
// per-node //line directives; the source-editing path edits in place), so each
// has its own source golden. Their *symbol tables*, by contrast, must be
// identical/
//
// Each subdirectory of testdata/golden/ is one case:
//
//	testdata/golden/<case>/input.go                       -- a small .go file exercising one structure
//	testdata/golden/<case>/expected.source_editing.golden -- source-editing instrumented output
//	testdata/golden/<case>/expected.ast_rewriting.golden  -- AST-rewriting instrumented output
//	testdata/golden/<case>/expected.sym.golden            -- the symbol table both must emit (normalized)
//
// Regenerate goldens after changing inputs or (intentionally) changing
// instrumentor output:
//
//	go test ./scanners/coverage/instrumentor -run TestInstrumentGolden -update

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/scanners/coverage/instrumentor"
	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/scanners/coverage/symboltable"

	qt "github.com/go-quicktest/qt"
)

var update = flag.Bool("update", false, "regenerate golden files")

// testShimPkg is the notifier import path the instrumentor adds. Fixed so
// the generated import line is deterministic across runs and machines.
const testShimPkg = "antithesis.notifier/instrumentation"

func TestMain(m *testing.M) {
	// Send logs to /dev/null to keep test output clean and avoid nil panics
	common.NewLogWriter(os.DevNull, common.Normal)
	os.Exit(m.Run())
}

type instrumenter interface {
	Instrument(path string) (string, error)
}

var backends = []struct {
	name   string
	golden string
	create func(basePath, shimPkg string, table *symboltable.SymbolTable) instrumenter
}{
	{
		name:   "source_editing",
		golden: "expected.source_editing.golden",
		create: func(b, s string, t *symboltable.SymbolTable) instrumenter {
			return instrumentor.CreateSourceEditingInstrumentor(b, s, t)
		},
	},
	{
		name:   "ast_rewriting",
		golden: "expected.ast_rewriting.golden",
		create: func(b, s string, t *symboltable.SymbolTable) instrumenter {
			return instrumentor.CreateInstrumentor(b, s, t)
		},
	},
}

func TestInstrumentGolden(t *testing.T) {
	inputs, err := filepath.Glob("testdata/golden/*/input.go")
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.IsTrue(len(inputs) > 0))

	for _, inputPath := range inputs {
		caseDir := filepath.Dir(inputPath)
		name := filepath.Base(caseDir)
		t.Run(name, func(t *testing.T) {
			symGolden := filepath.Join(caseDir, "expected.sym.golden")

			var symbols []string
			for _, be := range backends {
				res := runInstrument(t, inputPath, be.create)
				checkGolden(t, filepath.Join(caseDir, be.golden), res.source)
				checkGolden(t, symGolden, res.symbols)
				symbols = append(symbols, res.symbols)
			}
			qt.Check(t, qt.Equals(symbols[1], symbols[0]),
				qt.Commentf("%s and %s must emit identical symbol tables", backends[0].name, backends[1].name))
		})
	}
}

type instrumentResult struct {
	source  string // the instrumented source ("" if the file was skipped/copied)
	symbols string // the symbol table file, with non-deterministic bits normalized
}

func runInstrument(t *testing.T, inputPath string, create func(basePath, shimPkg string, table *symboltable.SymbolTable) instrumenter) instrumentResult {
	t.Helper()

	src, err := os.ReadFile(inputPath)
	qt.Assert(t, qt.IsNil(err))

	dir := t.TempDir()
	tmpInput := filepath.Join(dir, "input.go")
	qt.Assert(t, qt.IsNil(os.WriteFile(tmpInput, src, 0644)))

	symPath := filepath.Join(dir, "test.sym.tsv")
	sym, err := symboltable.CreateSymbolTableFile(symPath, "testmodule")
	qt.Assert(t, qt.IsNil(err))

	out, err := create(dir, testShimPkg, sym).Instrument(tmpInput)
	qt.Assert(t, qt.IsNil(err))

	// Flush the symbol table before reading it back.
	qt.Assert(t, qt.IsNil(sym.Close()))
	symBytes, err := os.ReadFile(symPath)
	qt.Assert(t, qt.IsNil(err))

	return instrumentResult{
		source:  out,
		symbols: normalizeSymbols(string(symBytes), tmpInput),
	}
}

var instrumentorLineRe = regexp.MustCompile(`(?m)^# instrumentor = .*$`)

func normalizeSymbols(s, tmpInput string) string {
	s = strings.ReplaceAll(s, tmpInput, "input.go")
	s = instrumentorLineRe.ReplaceAllString(s, "# instrumentor = INSTRUMENTOR")
	return s
}

// checkGolden compares got against the golden file, or (with -update) writes
// got to it.
func checkGolden(t *testing.T, goldenPath, got string) {
	t.Helper()

	if *update {
		qt.Assert(t, qt.IsNil(os.WriteFile(goldenPath, []byte(got), 0644)))
		return
	}

	want, err := os.ReadFile(goldenPath)
	if os.IsNotExist(err) {
		t.Fatalf("golden %s does not exist; run: go test ./... -run TestInstrumentGolden -update", goldenPath)
	}
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(got, string(want)))
}
