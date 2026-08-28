package instrumentor

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"

	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/scanners/coverage/symboltable"
)

// SourceEditingInstrumentor *is* the Antithesis Go source-code instrumentor.
type SourceEditingInstrumentor struct {
	fset        *token.FileSet
	symbolTable *symboltable.SymbolTable
	fullName    string
	shortName   string
	basePath    string
	shimPkg     string
	// the identifier the emitted calls are made on
	callbackName string
	funcStack    Stack
	nodeStack    Stack
	CurrentEdge  int
	src          []byte
	buf          *editBuffer
}

// CreateSourceEditingInstrumentor is the factory method.
func CreateSourceEditingInstrumentor(basePath string, shimPkg string, table *symboltable.SymbolTable) *SourceEditingInstrumentor {
	if len(basePath) > 0 && !strings.HasSuffix(basePath, "/") {
		basePath = basePath + "/"
	}
	return &SourceEditingInstrumentor{
		basePath:     basePath,
		fset:         token.NewFileSet(),
		shimPkg:      shimPkg,
		symbolTable:  table,
		callbackName: InstrumentationPackageAlias,
	}
}

func (instrumentor *SourceEditingInstrumentor) SetCallbackName(name string) {
	instrumentor.callbackName = name
}

// Edges returns the number of coverage edges recorded so far (across every file
// instrumented by this instance). It exposes CurrentEdge through the interface
// the coverage package selects implementations on.
func (instrumentor *SourceEditingInstrumentor) Edges() int {
	return instrumentor.CurrentEdge
}

// Instrument inserts calls to the Golang bridge to the Antithesis fuzzer.
// Errors should be logged, but are generally not fatal, since the input
// file can simply be copied to the output uninstrumented. If a file contains
// no executable code (i.e. contains only variable declarations, exports,
// or imports, an empty string is returned, so that the caller can simply
// copy the input file.
// TODO Return a * to a string, rather that returning the empty string to
// signal "I didn't instrument this input."
func (instrumentor *SourceEditingInstrumentor) Instrument(path string) (string, error) {
	bytes, e := os.ReadFile(path)
	if e != nil {
		return "", e
	}

	instrumentor.fullName = path
	instrumentor.shortName = strings.TrimPrefix(path, instrumentor.basePath)
	startingEdge := instrumentor.CurrentEdge

	f, e := parser.ParseFile(instrumentor.fset, path, bytes, parser.ParseComments)
	if e != nil {
		return "", e
	}

	if ExportsFunctions(f, instrumentor.fset) {
		common.Logger.Printf(common.Normal, "File %s exports functions, and will not be instrumented", path)
		return "", nil
	}

	if HasLinkname(f, instrumentor.fset) {
		common.Logger.Printf(common.Normal, "File %s exports linknames, and will not be instrumented", path)
		return "", nil
	}

	instrumentor.src = bytes
	instrumentor.buf = &editBuffer{src: bytes}
	ast.Walk(instrumentor, f)

	if instrumentor.CurrentEdge == startingEdge {
		common.Logger.Printf(common.Info, "File %s has no code to be instrumented, and will simply be copied", path)
		return "", nil
	}

	// Pull in the notifier shim inline (no extra line) so nothing shifts. An
	// empty shimPkg means the caller supplies the callback itself
	if instrumentor.shimPkg != "" {
		instrumentor.buf.insert(instrumentor.offAt(f.Name.End()),
			fmt.Sprintf("; import %s %q", instrumentor.callbackName, instrumentor.shimPkg))
	}

	// Prepend one //line directive (as cmd/cover does) because compilation will happen in a temporary directory
	if IsLineDirectiveCompatible(f, instrumentor.fset) {
		instrumentor.buf.insert(0, fmt.Sprintf("//line %s:1:1\n", instrumentor.shortName))
	} else {
		common.Logger.Printf(common.Normal, "File %s has functions which are incompatible with //line directives. Will be instrumented but not //line-annotated.", path)
	}
	return string(instrumentor.buf.bytes()), nil
}

func (instrumentor *SourceEditingInstrumentor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		if top, _ := instrumentor.nodeStack.Pop(); isFuncDecl(top) {
			instrumentor.funcStack.Pop()
		}
		return instrumentor
	}

	switch n := node.(type) {
	case *ast.FuncDecl:
		if n.Name.String() == "init" {
			// init runs regardless of what we do, so instrumenting it is noise.
			return nil
		}
	case *ast.GenDecl:
		if n.Tok != token.VAR {
			return nil // constants and types are not interesting
		}

	case *ast.BlockStmt:
		// If it's a switch or select, the body is a list of case clauses; don't tag the block itself.
		switch {
		case len(n.List) > 0 && isCaseClause(n.List[0]): // switch
			for _, s := range n.List {
				clause := s.(*ast.CaseClause)
				// span starts at the `case`/`default` keyword; Notify is inserted after the colon.
				instrumentor.instrumentEdge(clause.Pos(), clause.Colon, clause.End(), clause.Body, false)
			}
		case len(n.List) > 0 && isCommClause(n.List[0]): // select
			for _, s := range n.List {
				clause := s.(*ast.CommClause)
				instrumentor.instrumentEdge(clause.Pos(), clause.Colon, clause.End(), clause.Body, false)
			}
		default:
			instrumentor.instrumentEdge(n.Lbrace, n.Lbrace, n.Rbrace+1, n.List, true) // +1 to step past closing brace.
		}
	case *ast.IfStmt:
		if n.Init != nil {
			ast.Walk(instrumentor, n.Init)
		}
		if n.Cond != nil {
			ast.Walk(instrumentor, n.Cond)
		}
		ast.Walk(instrumentor, n.Body)
		switch els := n.Else.(type) {
		case nil:
			// Add else because we want coverage for "not taken".
			edge := instrumentor.recordEdge(n.Body.End(), n.Body.End()+1)
			instrumentor.buf.insert(instrumentor.offAt(n.Body.End()), " else {"+instrumentor.notifyCall(edge)+"}")
		case *ast.BlockStmt:
			// Start at end of the "if" block so the covered part looks like it starts at the "else".
			instrumentor.instrumentEdge(n.Body.End(), els.Lbrace, els.Rbrace+1, els.List, true)
			for _, s := range els.List {
				ast.Walk(instrumentor, s)
			}
		case *ast.IfStmt:
			// Start at end of the "if" block so the covered part looks like it starts at the "else".
			edge := instrumentor.recordEdge(n.Body.End(), instrumentor.statementBoundary(els))
			instrumentor.buf.insert(instrumentor.offAt(els.Pos()), "{"+instrumentor.notifyCall(edge)+"; ")
			// Walk before closing the wrapper: if the else-if has no trailing
			// else, walking synthesizes one at els.End(); the wrapper's closing
			// brace must be inserted AFTER that (same offset, so later insertion
			// wins the tie) or the synthesized else detaches from its if.
			ast.Walk(instrumentor, els)
			instrumentor.buf.insert(instrumentor.offAt(els.End()), "}")
		default:
			common.Logger.Fatalf("Unexpected node type in else: %v (%T)", n.Else, n.Else)
		}
		return nil
	case *ast.ForStmt:
		// TODO: handle increment statement
	case *ast.SelectStmt:
		// Don't annotate an empty select - creates a syntax error.
		if n.Body == nil || len(n.Body.List) == 0 {
			return nil
		}
	case *ast.SwitchStmt:
		// Don't annotate an empty switch - creates a syntax error.
		if n.Body == nil || len(n.Body.List) == 0 {
			return nil
		}
		// Walk the switch's own parts first so the real case clauses claim their
		// edge ids before the synthesized default.
		if n.Init != nil {
			ast.Walk(instrumentor, n.Init)
		}
		if n.Tag != nil {
			ast.Walk(instrumentor, n.Tag)
		}
		ast.Walk(instrumentor, n.Body)
		hasDefault := false
		for _, s := range n.Body.List {
			if cas, ok := s.(*ast.CaseClause); ok && cas.List == nil {
				hasDefault = true
				break
			}
		}
		if !hasDefault {
			// Synthesize a default case so "no case matched" is covered.
			edge := instrumentor.recordEdge(n.Body.Rbrace, n.Body.Rbrace+1)
			rbrace := instrumentor.offAt(n.Body.Rbrace)
			sep := ""
			if instrumentor.sameLineCodePrecedes(rbrace) {
				sep = "; "
			}
			instrumentor.buf.insert(rbrace, sep+"default: "+instrumentor.notifyCall(edge)+";")
		}
		return nil
	case *ast.TypeSwitchStmt:
		// TODO: add default case
		if n.Body == nil || len(n.Body.List) == 0 {
			return nil // annotating an empty type switch would be a syntax error
		}
	case *ast.BinaryExpr:
		if n.Op == token.LAND || n.Op == token.LOR {
			// Replace the whole (possibly nested) short-circuit expression in one
			// edit and prune the walk, so no other edit lands inside the span.
			instrumentor.instrumentShortCircuit(n)
			return nil
		}
	case *ast.BadExpr:
		common.Logger.Fatalf("Invalid input: %v (%T)", node, node)
	case *ast.BadDecl:
		common.Logger.Fatalf("Invalid input: %v (%T)", node, node)
	}

	// Returning a non-nil visitor lets ast.Walk descend into this node's
	// children; push it so the matching nil visit can unwind funcStack.
	if isFuncDecl(node) {
		instrumentor.funcStack.Push(node)
	}
	instrumentor.nodeStack.Push(node)
	return instrumentor
}

func isFuncDecl(n ast.Node) bool {
	_, ok := n.(*ast.FuncDecl)
	return ok
}

func isCaseClause(n ast.Node) bool {
	_, ok := n.(*ast.CaseClause)
	return ok
}

func isCommClause(n ast.Node) bool {
	_, ok := n.(*ast.CommClause)
	return ok
}

func (instrumentor *SourceEditingInstrumentor) statementBoundary(s ast.Stmt) token.Pos {
	// Control flow statements are easy.
	switch s := s.(type) {
	case *ast.BlockStmt:
		// Treat blocks like basic blocks to avoid overlapping counters.
		return s.Lbrace
	case *ast.IfStmt:
		found, pos := hasFuncLiteral(s.Init)
		if found {
			return pos
		}
		found, pos = hasFuncLiteral(s.Cond)
		if found {
			return pos
		}
		return s.Body.Lbrace
	case *ast.ForStmt:
		found, pos := hasFuncLiteral(s.Init)
		if found {
			return pos
		}
		found, pos = hasFuncLiteral(s.Cond)
		if found {
			return pos
		}
		found, pos = hasFuncLiteral(s.Post)
		if found {
			return pos
		}
		return s.Body.Lbrace
	case *ast.LabeledStmt:
		return instrumentor.statementBoundary(s.Stmt)
	case *ast.RangeStmt:
		found, pos := hasFuncLiteral(s.X)
		if found {
			return pos
		}
		return s.Body.Lbrace
	case *ast.SwitchStmt:
		found, pos := hasFuncLiteral(s.Init)
		if found {
			return pos
		}
		found, pos = hasFuncLiteral(s.Tag)
		if found {
			return pos
		}
		return s.Body.Lbrace
	case *ast.SelectStmt:
		return s.Body.Lbrace
	case *ast.TypeSwitchStmt:
		found, pos := hasFuncLiteral(s.Init)
		if found {
			return pos
		}
		return s.Body.Lbrace
	}
	found, pos := hasFuncLiteral(s)
	if found {
		return pos
	}
	return s.End()
}

func (instrumentor *SourceEditingInstrumentor) instrumentEdge(spanStart, lbrace, blockEnd token.Pos, list []ast.Stmt, extendToClosingBrace bool) {
	// Special case: cover an empty block by sitting just inside the braces. A
	// separator is still needed when real code follows on the same line.
	if len(list) == 0 {
		off := instrumentor.offAfter(lbrace)
		instrumentor.insertEdge(off, instrumentor.firstBlockSep(off), spanStart, blockEnd)
		return
	}
	// A statement list may hold several basic blocks due to statements that
	// affect the flow of control.
	pos := spanStart
	firstBlock := true
	for {
		// Find first statement that affects flow of control (break, continue, if, etc.).
		// It will be the last statement of this basic block.
		var last int
		end := blockEnd
		for last = 0; last < len(list); last++ {
			end = instrumentor.statementBoundary(list[last])
			if instrumentor.endsBasicSourceBlock(list[last]) {
				extendToClosingBrace = false // Block is broken up now.
				last++
				break
			}
		}
		if extendToClosingBrace {
			end = blockEnd
		}
		if pos != end { // Can have no source to cover if e.g. blocks abut.
			if firstBlock {
				// Just past the '{' / ':'. A ';' is added only when real code
				// follows on the same line (e.g. `func() { x() }`); otherwise a
				// newline or // comment lets auto-semicolon insertion do the job.
				off := instrumentor.offAfter(lbrace)
				instrumentor.insertEdge(off, instrumentor.firstBlockSep(off), pos, end)
			} else {
				// Inline, directly before this basic block's leading statement.
				instrumentor.insertEdge(instrumentor.offAt(list[0].Pos()), "; ", pos, end)
			}
		}
		list = list[last:]
		if len(list) == 0 {
			break
		}
		firstBlock = false
		pos = list[0].Pos()
	}
}

func (instrumentor *SourceEditingInstrumentor) endsBasicSourceBlock(s ast.Stmt) bool {
	switch s := s.(type) {
	case *ast.BlockStmt:
		// Treat blocks like basic blocks to avoid overlapping counters.
		return true
	case *ast.BranchStmt:
		return true
	case *ast.ForStmt:
		return true
	case *ast.IfStmt:
		return true
	case *ast.LabeledStmt:
		return instrumentor.endsBasicSourceBlock(s.Stmt)
	case *ast.RangeStmt:
		return true
	case *ast.SwitchStmt:
		return true
	case *ast.SelectStmt:
		return true
	case *ast.TypeSwitchStmt:
		return true
	case *ast.ExprStmt:
		// Calls to panic change the flow.
		// We really should verify that "panic" is the predefined function,
		// but without type checking we can't and the likelihood of it being
		// an actual problem is vanishingly small.
		if call, ok := s.X.(*ast.CallExpr); ok {
			if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "panic" && len(call.Args) == 1 {
				return true
			}
		}
	}
	found, _ := hasFuncLiteral(s)
	return found
}
