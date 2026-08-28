package instrumentor

import (
	"fmt"
	"go/ast"
	"go/token"
	"sort"

	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/scanners/coverage/symboltable"
)

type edit struct {
	start, end int // byte offsets into the original file
	text       string
}

type editBuffer struct {
	src   []byte
	edits []edit
}

func (b *editBuffer) insert(at int, text string) { b.edits = append(b.edits, edit{at, at, text}) }

func (b *editBuffer) replace(start, end int, text string) {
	b.edits = append(b.edits, edit{start, end, text})
}

func (b *editBuffer) bytes() []byte {
	sort.SliceStable(b.edits, func(i, j int) bool { return b.edits[i].start < b.edits[j].start })
	var out []byte
	prev := 0
	for _, e := range b.edits {
		out = append(out, b.src[prev:e.start]...)
		out = append(out, e.text...)
		prev = e.end
	}
	return append(out, b.src[prev:]...)
}

func (instrumentor *SourceEditingInstrumentor) offAt(p token.Pos) int {
	return instrumentor.fset.Position(p).Offset
}
func (instrumentor *SourceEditingInstrumentor) offAfter(p token.Pos) int {
	return instrumentor.fset.Position(p).Offset + 1
}

// Is there an unterminated statement without a newline preceding `off`?
func (instrumentor *SourceEditingInstrumentor) sameLineCodePrecedes(off int) bool {
	for i := off - 1; i >= 0; i-- {
		switch c := instrumentor.src[i]; {
		case c == ' ' || c == '\t':
			continue
		case c == '\n' || c == '\r':
			return false
		default:
			return true
		}
	}
	return false
}

func (instrumentor *SourceEditingInstrumentor) firstBlockSep(off int) string {
	for i := off; i < len(instrumentor.src); i++ {
		switch c := instrumentor.src[i]; {
		case c == ' ' || c == '\t':
			continue
		case c == '\n' || c == '\r':
			return ""
		case c == '/' && i+1 < len(instrumentor.src) && instrumentor.src[i+1] == '/':
			return ""
		default:
			return ";"
		}
	}
	return ""
}

func (instrumentor *SourceEditingInstrumentor) notifyCall(edge int) string {
	return fmt.Sprintf("%s.%s(%d)", instrumentor.callbackName, AntithesisCallbackFunction, edge)
}

func (instrumentor *SourceEditingInstrumentor) recordEdge(start, end token.Pos) int {
	instrumentor.CurrentEdge++

	s := instrumentor.fset.Position(start)
	e := instrumentor.fset.Position(end)
	fname := ""
	if maybeDecl, ok := instrumentor.funcStack.Peek(); ok {
		if decl, isDecl := maybeDecl.(*ast.FuncDecl); isDecl {
			fname = decl.Name.Name
		}
	}

	if err := instrumentor.symbolTable.WritePosition(symboltable.SymbolTablePosition{
		Path:        instrumentor.fullName,
		Function:    fname,
		StartLine:   s.Line,
		StartColumn: s.Column,
		EndLine:     e.Line,
		EndColumn:   e.Column,
		Edge:        instrumentor.CurrentEdge,
	}); err != nil {
		common.Logger.Fatalf("Could not write symbol table line: %s", err.Error())
	}
	return instrumentor.CurrentEdge
}

func (instrumentor *SourceEditingInstrumentor) insertEdge(insertOff int, sep string, start, end token.Pos) {
	edge := instrumentor.recordEdge(start, end)
	instrumentor.buf.insert(insertOff, instrumentor.notifyCall(edge)+sep)
}

func (instrumentor *SourceEditingInstrumentor) instrumentShortCircuit(n *ast.BinaryExpr) {
	// Walk the left operand first so operands nested inside it are numbered
	// before the outer operand that sits to their right
	ast.Walk(instrumentor, n.X)

	end := n.Y.End()
	if found, pos := hasFuncLiteral(n.Y); found {
		end = pos
	}
	edge := instrumentor.recordEdge(n.Y.Pos(), end)
	instrumentor.buf.insert(instrumentor.offAt(n.Y.Pos()),
		fmt.Sprintf("func() bool { %s; return (", instrumentor.notifyCall(edge)))
	instrumentor.buf.insert(instrumentor.offAt(n.Y.End()), ") == true }() == true")

	// Walk the right operand (left in place, at its original positions) so its
	// own subtree gets instrumented.
	ast.Walk(instrumentor, n.Y)
}
