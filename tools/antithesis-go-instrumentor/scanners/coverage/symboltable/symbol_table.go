package symboltable

import (
	"bufio"
	"fmt"
	"os"

	"github.com/antithesishq/antithesis-sdk-go/internal"
)

// SymbolTable is the serialization of the
// edges that the instrumentor finds and
// instruments.
type SymbolTable struct {
	Path   string
	writer *fileSymbolTableWriter
}

type SymbolTablePosition struct {
	Path        string
	Class       string // the receiver type of a method; "" for a plain function
	Function    string
	StartLine   int
	StartColumn int
	EndLine     int
	EndColumn   int
	Edge        int
}

// CreateSymbolTableFile opens an Antithesis-standard .sym.tsv file on disk.
// instrumentedModule is the key the runtime registers the file under;
// displayName is the label shown to people (see the "# name" global in
// instrumentation/symbol_file_format.md).
func CreateSymbolTableFile(symbolTablePath, instrumentedModule, displayName string) (symbolTable *SymbolTable, err error) {
	var w *fileSymbolTableWriter
	if w, err = createFileSymbolTableWriter(symbolTablePath); err != nil {
		return
	}

	symbolTable = &SymbolTable{
		Path:   symbolTablePath,
		writer: w,
	}
	if err = symbolTable.writeHeader(instrumentedModule, displayName); err != nil {
		symbolTable = nil
	}
	return
}

// writeHeader writes the Antithesis-standard symbol table header.
func (t *SymbolTable) writeHeader(module, displayName string) error {
	for _, line := range []string{
		"# language = Go",
		"# instrumentor = antithesis-go-instrumentor " + internal.SDK_Version,
		"# module = " + module,
		"# name = " + displayName,
		"file\tclass\tfunction\tbegin_line\tbegin_column\tend_line\tend_column\taddress",
	} {
		if err := t.writer.WriteLine(line); err != nil {
			return err
		}
	}
	return nil
}

// WritePosition describes a callback to the Antithesis instrumentation.
func (t *SymbolTable) WritePosition(p SymbolTablePosition) error {
	line := fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d", p.Path, p.Class, p.Function, p.StartLine, p.StartColumn, p.EndLine, p.EndColumn, p.Edge)
	return t.writer.WriteLine(line)
}

// Close closes the underlying file resources.
func (t *SymbolTable) Close() error {
	return t.writer.Close()
}

// ---------------------------------------------------------------------------------
// fileSymbolTableWriter
// ---------------------------------------------------------------------------------

type fileSymbolTableWriter struct {
	f      *os.File
	writer *bufio.Writer
}

func createFileSymbolTableWriter(name string) (symWriter *fileSymbolTableWriter, err error) {
	var f *os.File
	if f, err = os.Create(name); err != nil {
		return
	}
	symWriter = &fileSymbolTableWriter{
		f:      f,
		writer: bufio.NewWriter(f),
	}
	return
}

func (w *fileSymbolTableWriter) WriteLine(s string) error {
	_, err := w.writer.WriteString(s + "\n")
	if err != nil {
		return err
	}
	return w.writer.Flush()
}

func (w *fileSymbolTableWriter) Close() error {
	err := w.writer.Flush()
	if err != nil {
		return err
	}
	return w.f.Close()
}
