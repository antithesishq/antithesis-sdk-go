package symboltable

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	qt "github.com/go-quicktest/qt"
)

func TestCreateSymbolTableFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.symbols.tsv")

	st, err := CreateSymbolTableFile(path, "mymodule")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(st.Path, path))
	st.Close()

	content, err := os.ReadFile(path)
	qt.Assert(t, qt.IsNil(err))

	lines := strings.Split(strings.TrimRight(string(content), "\n"), "\n")
	qt.Assert(t, qt.HasLen(lines, 4))
	qt.Check(t, qt.Equals(lines[0], "# language = Go"))
	qt.Check(t, qt.StringContains(lines[1], "# instrumentor = "))
	qt.Check(t, qt.Equals(lines[2], "# module = mymodule"))
	qt.Check(t, qt.Equals(lines[3], "file\tfunction\tbegin_line\tbegin_column\tend_line\tend_column\taddress"))
}

func TestWritePosition(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.symbols.tsv")

	st, err := CreateSymbolTableFile(path, "mymodule")
	qt.Assert(t, qt.IsNil(err))

	err = st.WritePosition(SymbolTablePosition{
		Path:        "/src/main.go",
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
	// 4 header lines + 1 position line
	qt.Assert(t, qt.HasLen(lines, 5))
	qt.Check(t, qt.Equals(lines[4], "/src/main.go\tFoo\t10\t1\t15\t2\t42"))
}
