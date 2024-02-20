package instrumentor

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// SymbolTable is the serialization of the
// edges that the instrumentor finds and
// instruments.
type SymbolTable struct {
	Path       string
	writer     symbolTableWriter
	executable string
}

type SymbolTablePosition struct {
	Path        string
	Function    string
	StartLine   int
	StartColumn int
	EndLine     int
	EndColumn   int
	Edge        int
}

// CreateSymbolTableFile opens an Antithesis-standard .symbols.tsv file on disk.
func CreateSymbolTableFile(symbolTablePath, instrumentedModule string) (err error, symbolTable *SymbolTable) {

  var w *fileSymbolTableWriter
	if err, w = createFileSymbolTableWriter(symbolTablePath); err != nil {
    return
  }

	// There can be an error if the file has been moved!
	executable, _ := os.Executable()
	symbolTable = &SymbolTable{
    Path: symbolTablePath, 
    writer: w, 
    executable: executable,
  }
	if err = symbolTable.writeHeader(instrumentedModule); err != nil {
    symbolTable = nil
	}
	return
}

// CreateInMemorySymbolTable creates an in memory symbol table for testing.
func CreateInMemorySymbolTable(symbolTablePath, instrumentedModule string) *SymbolTable {
	w := createInMemorySymbolTableWriter()
	symbolTable := &SymbolTable{Path: symbolTablePath, writer: w, executable: "goinstrumentor"}
	symbolTable.writeHeader(instrumentedModule)
	return symbolTable
}

// WriteHeader writes the Antithesis-standard symbol table header.
func (t *SymbolTable) writeHeader(module string) error {
	if err := t.writer.WriteLine("# language = Go"); err != nil {
		return err
	}
	if err := t.writer.WriteLine("# instrumentor = " + t.executable); err != nil {
		return err
	}
	if err := t.writer.WriteLine("# module = " + module); err != nil {
		return err
	}
	return t.writer.WriteLine("file\tfunction\tbegin_line\tbegin_column\tend_line\tend_column\taddress")
}

// WritePosition describes a callback to the Antithesis instrumentation.
func (t *SymbolTable) WritePosition(p SymbolTablePosition) error {
	line := fmt.Sprintf("%s\t%s\t%d\t%d\t%d\t%d\t%d", p.Path, p.Function, p.StartLine, p.StartColumn, p.EndLine, p.EndColumn, p.Edge)
	return t.writer.WriteLine(line)
}

// Close closes the underlying file resources.
func (t *SymbolTable) Close() error {
	return t.writer.Close()
}

func (t *SymbolTable) String() string {
	return t.writer.String()
}

// --------------------------------------------------------------------------------
// symbolTableWriter
// --------------------------------------------------------------------------------
type symbolTableWriter interface {
	WriteLine(s string) error
	Close() error
	String() string
}

type fileSymbolTableWriter struct {
	f      *os.File
	writer *bufio.Writer
}

type inMemorySymbolTableWriter struct {
	writer strings.Builder
}


// --------------------------------------------------------------------------------
// fileSymbolTableWriter
// --------------------------------------------------------------------------------
func createFileSymbolTableWriter(name string) (err error, symWriter *fileSymbolTableWriter) {
  var f *os.File
	if f, err = os.Create(name); err != nil {
    return
	}
  symWriter = &fileSymbolTableWriter{
    f: f,
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

func (fileSymbolTableWriter) String() string {
	// logger.Fatalf("fileSymbolTableWriter does not support String method")
	return ""
}


// --------------------------------------------------------------------------------
// inMemorySymbolTableWriter
// --------------------------------------------------------------------------------
func createInMemorySymbolTableWriter() symbolTableWriter {
	return &inMemorySymbolTableWriter{}
}

func (w *inMemorySymbolTableWriter) WriteLine(s string) error {
	_, err := w.writer.WriteString(s + "\n")
	return err
}

func (w *inMemorySymbolTableWriter) Close() error {
	return nil
}

func (w *inMemorySymbolTableWriter) String() string {
	return w.writer.String()
}
