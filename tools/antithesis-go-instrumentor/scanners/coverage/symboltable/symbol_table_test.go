package symboltable

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/antithesishq/antithesis-sdk-go/internal"
	qt "github.com/go-quicktest/qt"
)

func TestCreateSymbolTableFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.symbols.tsv")

	st, err := CreateSymbolTableFile(path, "mymodule", "example.com/mymodule")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(st.Path, path))
	st.Close()

	content, err := os.ReadFile(path)
	qt.Assert(t, qt.IsNil(err))

	lines := strings.Split(strings.TrimRight(string(content), "\n"), "\n")
	qt.Assert(t, qt.HasLen(lines, 5))
	qt.Check(t, qt.Equals(lines[0], "# language = Go"))
	qt.Check(t, qt.Equals(lines[1], "# instrumentor = antithesis-go-instrumentor "+internal.SDK_Version))
	// The contract: a tool name of lowercase letters, digits and hyphens, then
	// optionally one space and a version. Never a path.
	qt.Check(t, qt.Matches(lines[1], `# instrumentor = [a-z0-9-]+( [^ ]+)?`))
	qt.Check(t, qt.Equals(lines[2], "# module = mymodule"))
	qt.Check(t, qt.Equals(lines[3], "# name = example.com/mymodule"))
	// Consumers resolve columns by name, so the header's order is free; what
	// it must name is the full set.
	qt.Check(t, qt.DeepEquals(columnSet(lines[4]), columnSet("file\tclass\tfunction\tbegin_line\tbegin_column\tend_line\tend_column\taddress")))
}

func columnSet(header string) map[string]bool {
	set := map[string]bool{}
	for _, column := range strings.Split(header, "\t") {
		set[column] = true
	}
	return set
}

func TestWritePosition(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.symbols.tsv")

	st, err := CreateSymbolTableFile(path, "mymodule", "example.com/mymodule")
	qt.Assert(t, qt.IsNil(err))

	err = st.WritePosition(SymbolTablePosition{
		Path:        "/src/main.go",
		Class:       "Reader",
		Function:    "Foo",
		StartLine:   10,
		StartColumn: 1,
		EndLine:     15,
		EndColumn:   2,
		Edge:        42,
	})
	qt.Assert(t, qt.IsNil(err))
	st.Close()

	content, err := os.ReadFile(path)
	qt.Assert(t, qt.IsNil(err))

	lines := strings.Split(strings.TrimRight(string(content), "\n"), "\n")
	// 5 header lines + 1 position line
	qt.Assert(t, qt.HasLen(lines, 6))
	row := map[string]string{}
	for i, column := range strings.Split(lines[4], "\t") {
		row[column] = strings.Split(lines[5], "\t")[i]
	}
	qt.Check(t, qt.DeepEquals(row, map[string]string{
		"file": "/src/main.go", "class": "Reader", "function": "Foo",
		"begin_line": "10", "begin_column": "1", "end_line": "15", "end_column": "2", "address": "42",
	}))
}
