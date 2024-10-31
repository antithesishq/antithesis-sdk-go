package instrumentor

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
	"golang.org/x/tools/go/ast/astutil"
)

// InstrumentationPackageAlias will be used to prevent any collisions
// between possible other packages named "instrumentation". Underscore
// characters are considered bad style, which is why I'm using them:
// a collision is less likely.
const InstrumentationPackageAlias = "__antithesis_instrumentation__"

// AntithesisCallbackFunction is the name of the instrumentor-generated
// callback function that delegates to wrapper.Notify() with the correct
// arguments. Multiple definitions of this function will lead to a (desirable)
// compile-time error.
const AntithesisCallbackFunction = "Notify"

var compilationRelevantCommentRegex, _ = regexp.Compile(`(?sm)^\s*(go:|\+build)`)

// Capitalized struct items are accessed outside this file
type CoverageInstrumentor struct {
	GoInstrumentor    *Instrumentor
	SymTable          *SymbolTable
	logWriter         *common.LogWriter
	UsingSymbols      string
	FullCatalogPath   string
	NotifierPackage   string
	PreviousEdge      int
	FilesInstrumented int
	FilesSkipped      int
}

type NotifierInfo struct {
	logWriter                  *common.LogWriter
	InstrumentationPackageName string
	SymbolTableName            string
	NotifierPackage            string
	EdgeCount                  int
}

func (cI *CoverageInstrumentor) WriteNotifierSource(notifierDir string, edge_count int) {
	if cI.GoInstrumentor == nil {
		return
	}

	notifierInfo := NotifierInfo{
		InstrumentationPackageName: common.InstrumentationPackageName(),
		SymbolTableName:            cI.UsingSymbols,
		EdgeCount:                  edge_count,
		NotifierPackage:            cI.NotifierPackage,
		logWriter:                  common.GetLogWriter(),
	}

	GenerateNotifierSource(notifierDir, &notifierInfo)
}

func (cI *CoverageInstrumentor) InstrumentFile(file_name string) string {
	if cI.GoInstrumentor == nil {
		return ""
	}
	if cI.logWriter == nil {
		cI.logWriter = common.GetLogWriter()
	}
	var err error
	instrumented := ""
	cI.logWriter.Printf("Instrumenting %s", file_name)
	cI.PreviousEdge = cI.GoInstrumentor.CurrentEdge
	if instrumented, err = cI.GoInstrumentor.Instrument(file_name); err != nil {
		cI.logWriter.Printf("Error: File %s produced error %s; simply copying source", file_name, err)
		return ""
	}

	return instrumented
}

func (cI *CoverageInstrumentor) WrapUp() (edge_count int) {
	var err error
	edge_count = 0

	if cI.logWriter == nil {
		cI.logWriter = common.GetLogWriter()
	}
	if cI.GoInstrumentor != nil {
		if err = cI.SymTable.Close(); err != nil {
			cI.logWriter.Printf("Error Could not close symbol table %s: %s", cI.SymTable.Path, err)
		}
		cI.logWriter.Printf("Symbol table: %s", cI.SymTable.Path)
		edge_count = cI.GoInstrumentor.CurrentEdge
	}
	return
}

func (cI *CoverageInstrumentor) SummarizeWork(numFiles int) {
	if cI.GoInstrumentor == nil {
		return
	}
	if cI.logWriter == nil {
		cI.logWriter = common.GetLogWriter()
	}

	numFilesSkipped := (numFiles - cI.FilesInstrumented) + cI.FilesSkipped
	numEdges := cI.GoInstrumentor.CurrentEdge
	cI.logWriter.Printf("%d '.go' %s instrumented, %d %s skipped, %d %s identified",
		numFiles, common.Pluralize(numFiles, "file"),
		numFilesSkipped, common.Pluralize(numFilesSkipped, "file"),
		numEdges, common.Pluralize(numEdges, "edge"))
}

// IsFunctionExported checks the comments preceding a function declaration
// for all known formats of export directive.
func IsFunctionExported(group *ast.CommentGroup, name string) bool {
	if group == nil {
		return false
	}
	// No characters may precede or follow the directive.
	exportDeclaration := "//export " + name
	for _, comment := range group.List {
		if comment.Text == exportDeclaration {
			return true
		}
	}
	return false
}

// ExportsFunctions warns the caller that the the .go file includes
// export directives in comments, which AST-rewriting may damage.
func ExportsFunctions(file *ast.File, fset *token.FileSet) bool {
	foundExport := false
	finder := func(cursor *astutil.Cursor) bool {
		n := cursor.Node()
		switch n := n.(type) {
		case *ast.FuncDecl:
			if IsFunctionExported(n.Doc, n.Name.Name) {
				foundExport = true
				// Stop recursion.
				return false
			}
		}
		// By default, continue recursing.
		return true
	}

	astutil.Apply(file, finder, nil)
	return foundExport
}

// HasLinkname lets us exclude .go files that interact with other
// languages.
func HasLinkname(file *ast.File, fset *token.FileSet) bool {
	foundLinkname := false
	finder := func(cursor *astutil.Cursor) bool {
		n := cursor.Node()
		switch n := n.(type) {
		case *ast.FuncDecl:
			if n.Doc != nil {
				for _, comment := range n.Doc.List {
					if strings.Contains(comment.Text, "go:linkname") {
						foundLinkname = true
						return false
					}
				}
			}
		}
		return true
	}

	astutil.Apply(file, finder, nil)
	return foundLinkname
}

// Check to see if this particular node represents a something which requires
// runtime-generated file names. If this is the case, we can't instrument this
// because we have to statically set the path in the comments, and there's no way
// to simultaneously:
//
// 1) Use //line directives to set the line numbers; and
// 2) Let the runtime set the absolute/relative file path.
//
// The primary offenders are the runtime.(Caller|Callers) functions. See
// https://github.com/golang/go/issues/26207
// for more details.
func RequiresFileNameOrLineNumber(n ast.Node, fset *token.FileSet) bool {
	if n == nil {
		return false
	}
	call, callOk := n.(*ast.CallExpr)
	if !callOk {
		return false
	}
	if call.Fun == nil {
		return false
	}
	sel, selOk := call.Fun.(*ast.SelectorExpr)
	if !selOk {
		return false
	}
	if sel.X == nil || sel.Sel == nil {
		return false
	}
	x, xOk := sel.X.(*ast.Ident)
	if !xOk {
		return false
	}
	return x.Name == "runtime" && strings.Contains(sel.Sel.Name, "Caller")
}

func IsLineDirectiveCompatible(file *ast.File, fset *token.FileSet) bool {
	requiresFileNameOrLineNumber := false
	finder := func(cursor *astutil.Cursor) bool {
		n := cursor.Node()
		if RequiresFileNameOrLineNumber(n, fset) {
			requiresFileNameOrLineNumber = true
			return false
		}
		return true
	}

	astutil.Apply(file, finder, nil)
	return !requiresFileNameOrLineNumber
}

// The parser will strip compiler directives from CommentGroup.Text(),
// so we need a separate loop to look for them.
func commentContainsDirective(group *ast.CommentGroup) bool {
	for _, comment := range group.List {
		c := comment.Text
		//-style comment (no newline at the end)
		if c[1] == '/' {
			c = c[2:]
			if isDirective(c) {
				return true
			}
		}
	}
	// TODO It's possible that the above code suffices, but we can't know
	// that without more investigation.
	return compilationRelevantCommentRegex.MatchString(group.Text())
}

func runeInclusive(r, from, to rune) bool {
	if r < from || r > to {
		return false
	}
	return true
}

// From go/ast/ast.go
func isDirective(c string) bool {
	// TODO: Not sure if line directives may affect instrumentation so excluding
	// such comments for now
	if strings.HasPrefix(c, "line ") {
		return false
	}

	// "//line " is a line directive.
	// "//extern " is for gccgo.
	// "//export " is for cgo.
	// (The // has been removed.)
	if strings.HasPrefix(c, "line ") || strings.HasPrefix(c, "extern ") || strings.HasPrefix(c, "export ") {
		return true
	}

	// "//[a-z0-9]+:[a-z0-9]"
	// (The // has been removed.)
	colon := strings.Index(c, ":")
	if colon <= 0 || colon+1 >= len(c) {
		return false
	}
	for i := 0; i <= colon+1; i++ {
		if i == colon {
			continue
		}
		b := c[i]

		isValidRune := runeInclusive(rune(b), 'a', 'z') || runeInclusive(rune(b), '0', '9')
		if !isValidRune {
			return false
		}
	}
	return true
}

// Create a //line directive that the caller can add to the comments for the file/maybe associated with the Node.
// Note that we can insert these even between go:embed directives and the variables into which Go will embed
// the resource: https://pkg.go.dev/embed#hdr-Directives
// "Only blank lines and ‘//’ line comments are permitted between the directive and the declaration."
func (instrumentor *Instrumentor) createLineDirective(lineNumber int, node *ast.Node) *ast.CommentGroup {
	file := instrumentor.fset.File((*node).Pos())
	p := instrumentor.fset.Position((*node).Pos())
	currLine := p.Line
	if currLine == 1 {
		instrumentor.logWriter.Printf("Skipping inserting line position comment at very start of file")
		return nil
	}
	lineStartPos := file.LineStart(p.Line)
	if instrumentor.logWriter.VerboseLevel(2) {
		instrumentor.logWriter.Printf("Creating comment for node @ %v start of line %d at %v", (*node).Pos(), p.Line, lineStartPos)
	}
	if (*node).Pos() == lineStartPos {
		// This node is actually at the start of the line, so move the position back one to make sure there's no conflict.
		// Also, set the original lineStartPos to true in the map just to make sure we don't have another item on the same
		// line just re-create the problem.
		instrumentor.posLines[lineStartPos] = true
		lineStartPos--
	}
	if _, ok := instrumentor.posLines[lineStartPos]; ok {
		// If we've already dropped a //line directive at this position in the file, don't create another one.
		return nil
	}
	// Tag that we've created a //line directive for this spot in the file.
	instrumentor.posLines[lineStartPos] = true
	newComment := ast.Comment{Text: fmt.Sprintf("\n//line %s:%d", instrumentor.shortName, int(lineNumber)), Slash: lineStartPos}
	commentGroup := ast.CommentGroup{List: []*ast.Comment{&newComment}}
	return &commentGroup
}

// TrimComments uses the CommentMap structure to discard
// all comments not relevant to compilation.
func (instrumentor *Instrumentor) TrimComments(path string, file *ast.File) {
	var commentGroups []*ast.CommentGroup
	commentMap := ast.NewCommentMap(instrumentor.fset, file, file.Comments)
	// We can't iterate over this hash map, because we need to
	// encounter the comments in file order. So we'll walk the
	// AST once.
	stripper := func(cursor *astutil.Cursor) bool {
		node := cursor.Node()
		groups := commentMap[node]

		switch n := node.(type) {
		// Node types with no comments can be skipped.
		// Only a few node types may retain their comments,
		// in case they have go: or CGO directives.
		case *ast.AssignStmt:
		case *ast.BasicLit:
		case *ast.Comment:
		case *ast.CommentGroup:
		case *ast.DeclStmt:
		case *ast.Field:
			n.Doc = nil
			n.Comment = nil
		case *ast.ExprStmt:
		case *ast.File:
			// This applies to +build directives.
			// See https://golang.org/cmd/go/#hdr-Build_constraints
			// Package documentation appears in the Doc field *and*
			// in the Comments field.
			for _, group := range groups {
				if commentContainsDirective(group) {
					commentGroups = append(commentGroups, group)
				}
			}
		case *ast.ForStmt:
		case *ast.FuncDecl:
			text := n.Doc.Text()
			if strings.Contains(text, "go:") {
				p := instrumentor.fset.Position(n.Pos())
				instrumentor.logWriter.Printf("Warning: Function %s, file %s, line %d uses a go: directive. This file may have to be excluded from instrumentation.", n.Name.Name, path, p.Line)
			}
			n.Doc = nil
		case *ast.GenDecl:
			if n.Tok == token.IMPORT && n.Doc != nil && len(n.Specs) == 1 {
				// import "C" will have len(Specs) == 1
				spec := n.Specs[0].(*ast.ImportSpec)
				if spec.Path.Value == "\"C\"" {
					p := instrumentor.fset.Position(spec.Pos())
					instrumentor.logWriter.Printf("Warning: File %s, line %d imports a C declaration. This file may have to be excluded from instrumentation.", path, p.Line)
					commentGroups = append(commentGroups, n.Doc)
				}
			} else {
				n.Doc = nil
				// This code exists for the sake of go:embed.
				for _, group := range groups {
					if commentContainsDirective(group) {
						commentGroups = append(commentGroups, group)
					}
				}
			}
		case *ast.Ident:
		case *ast.IfStmt:
		case *ast.ImportSpec:
			n.Doc = nil
			n.Comment = nil
		case *ast.LabeledStmt:
		case *ast.RangeStmt:
		case *ast.ReturnStmt:
		case *ast.ValueSpec:
			// This code exists for the sake of go:embed.
			for _, group := range groups {
				if commentContainsDirective(group) {
					commentGroups = append(commentGroups, group)
				}
			}
		default:
			if instrumentor.logWriter.VerboseLevel(2) {
				instrumentor.logWriter.Printf("No comment revision for AST node of type %T\n", node)
			}
		}
		return true
	}
	// We don't need to output; we simply mutated the input AST's comments.
	astutil.Apply(file, stripper, nil)
	file.Comments = commentGroups
}

type functionLiteralFinder token.Pos

func (f *functionLiteralFinder) Visit(node ast.Node) (w ast.Visitor) {
	if f.found() {
		return nil // Prune search.
	}
	switch n := node.(type) {
	case *ast.FuncLit:
		*f = functionLiteralFinder(n.Body.Lbrace)
		return nil // Prune search.
	}
	return f
}

func (f *functionLiteralFinder) found() bool {
	return token.Pos(*f) != token.NoPos
}

func hasFuncLiteral(n ast.Node) (bool, token.Pos) {
	if n == nil {
		return false, 0
	}
	var literal functionLiteralFinder
	ast.Walk(&literal, n)
	return literal.found(), token.Pos(literal)
}

// Instrumentor *is* the Antithesis Go source-code instrumentor.
type Instrumentor struct {
	nodeLines   map[string]int
	posLines    map[token.Pos]bool
	logWriter   *common.LogWriter
	fset        *token.FileSet
	SymbolTable *SymbolTable
	typeCounts  map[string]int
	fullName    string
	pkg         string
	shortName   string
	basePath    string
	ShimPkg     string
	funcStack   Stack
	currPos     []string
	comments    []*ast.CommentGroup
	nodeStack   Stack
	CurrentEdge int
	addLines    bool
}

// CreateInstrumentor is the factory method.
func CreateInstrumentor(basePath string, shimPkg string, table *SymbolTable) *Instrumentor {
	if len(basePath) > 0 {
		basePath = basePath + "/"
	}
	result := &Instrumentor{
		basePath:    basePath,
		fset:        token.NewFileSet(),
		ShimPkg:     shimPkg,
		SymbolTable: table,
		typeCounts:  map[string]int{},
		nodeLines:   map[string]int{},
		currPos:     make([]string, 0),
		posLines:    map[token.Pos]bool{},
		logWriter:   common.GetLogWriter(),
	}
	return result
}

// TODO: (justin.moore) See if we can get away with just re-parsing the file in-memory.
func (instrumentor *Instrumentor) writeSource(source, path string) error {
	// Any errors here are fatal anyway, so I'm not checking.
	f, e := os.Create(path)
	if e != nil {
		instrumentor.logWriter.Printf("Warning: Could not create %s", path)
		return e
	}
	defer f.Close()
	_, e = f.WriteString(source)
	if e != nil {
		instrumentor.logWriter.Printf("Warning: Could not write instrumented source to %s", path)
		return e
	}
	if instrumentor.logWriter.VerboseLevel(1) {
		instrumentor.logWriter.Printf("Wrote instrumented output to %s", path)
	}
	return nil
}

// Instrument inserts calls to the Golang bridge to the Antithesis fuzzer.
// Errors should be logged, but are generally not fatal, since the input
// file can simply be copied to the output uninstrumented. If a file contains
// no executable code (i.e. contains only variable declarations, exports,
// or imports, an empty string is returned, so that the caller can simply
// copy the input file.
// TODO Return a * to a string, rather that returning the empty string to
// signal "I didn't instrument this input."
func (instrumentor *Instrumentor) Instrument(path string) (string, error) {
	var bytes []byte
	var e error
	var f *ast.File
	if bytes, e = os.ReadFile(path); e != nil {
		return "", e
	}

	instrumentor.fullName = path
	instrumentor.shortName = strings.TrimPrefix(path, instrumentor.basePath)
	startingEdge := instrumentor.CurrentEdge

	sourceCode := string(bytes)
	if f, e = parser.ParseFile(instrumentor.fset, path, sourceCode, parser.ParseComments); e != nil {
		return "", e
	}

	if ExportsFunctions(f, instrumentor.fset) {
		instrumentor.logWriter.Printf("File %s exports functions, and will not be instrumented", path)
		return "", nil
	}

	if HasLinkname(f, instrumentor.fset) {
		instrumentor.logWriter.Printf("File %s exports linknames, and will not be instrumented", path)
		return "", nil
	}

	instrumentor.TrimComments(path, f)

	// The first pass over the code. We're not adding lines, we're inserting the instrumentation callbacks and
	// taking note of where the various ast.Node objects are in the file.
	instrumentor.addLines = false
	instrumentor.comments = f.Comments
	instrumentor.resetTypeCounts(true)
	ast.Walk(instrumentor, f)
	f.Comments = instrumentor.comments

	if instrumentor.CurrentEdge == startingEdge {
		if instrumentor.logWriter.VerboseLevel(1) {
			instrumentor.logWriter.Printf("File %s has no code to be instrumented, and will simply be copied", path)
		}
		return "", nil
	}

	// If there's something in here which requires either file names or line directives to be set
	// at runtime (or otherwise is incompatible with static file/line directives), instrument but
	// do not add line annotations.
	if !IsLineDirectiveCompatible(f, instrumentor.fset) {
		instrumentor.logWriter.Printf("File %s has functions which are incompatible with //line directives. Will be instrumented but not //line-annotated.", path)
		// Note that we actually insert the instrumentation callback here.
		if sourceCode, e = instrumentor.formatInstrumentedAst(path, f, true); e != nil {
			return "", e
		}
		return sourceCode, nil
	}

	// Write the new AST out to a temp file on disk. This means that when we re-parse the file, it will
	// look like a completely new file and we don't have to worry about any state carrying through from
	// one parse to another. However, do not add in the import shim (the final 'false' parameter).
	iPath := path + ".instrumented-only.go"
	if sourceCode, e = instrumentor.formatInstrumentedAst(iPath, f, false); e != nil {
		return "", e
	}
	instrumentor.writeSource(sourceCode, iPath)
	if f, e = parser.ParseFile(instrumentor.fset, iPath, sourceCode, parser.ParseComments); e != nil {
		os.Remove(iPath)
		return "", e
	}

	// The second pass through the file. Add in the line directives. Reset the type counts, since we've
	// cached the mapping of node identifiers to the line on which they were originally placed.
	instrumentor.addLines = true
	instrumentor.comments = f.Comments
	instrumentor.resetTypeCounts(false)
	ast.Walk(instrumentor, f)
	sortComments(instrumentor.comments)
	f.Comments = instrumentor.comments

	// Create a string version of the final AST, and add in the shim (the final 'true' param).
	if sourceCode, e := instrumentor.formatInstrumentedAst(path, f, true); e == nil {
		os.Remove(iPath)
		return sourceCode, nil
	}
	// TODO What are we doing with the error value above?
	os.Remove(iPath)
	return "", nil
}

func (instrumentor *Instrumentor) resetTypeCounts(full bool) {
	instrumentor.typeCounts = map[string]int{}
	instrumentor.currPos = make([]string, 0)
	if full {
		// Clear out any state from any previous files.
		instrumentor.nodeLines = map[string]int{}
	}
}

func (instrumentor *Instrumentor) pushType(node ast.Node) {
	t := fmt.Sprintf("%T", node)
	count, ok := instrumentor.typeCounts[t]
	if !ok {
		count = 0
	}
	// This will create an identifier of the form ${nodeType}@${index}, indicating the type of the node
	// and the number of other nodes in the AST which have had this type. E.g.,
	// - *ast.AssignStmt@3 (the 4th assignment statement)
	// - *ast.Ident@24 (the 25th identifier)
	ts := fmt.Sprintf("%s@%d", t, count)
	// Append to the depth-first list of nodes we've traversed to get here.
	instrumentor.currPos = append(instrumentor.currPos, ts)
	instrumentor.typeCounts[t] = count + 1
}

func (instrumentor *Instrumentor) popType() {
	if len(instrumentor.currPos) == 0 {
		return
	}
	instrumentor.currPos = instrumentor.currPos[:len(instrumentor.currPos)-1]
}

// Get a string representing the current position in the AST, using the node identifiers defined above. E.g.,
//
//   - *ast.File@0|*ast.FuncDecl@0|*ast.BlockStmt@0|*ast.AssignStmt@3
//     this file |  1st function |  1st fn block  | 4th assignment statement in the file
//
// This allows us to uniquely identify any node in the AST based on a deterministic depth-first search going
// from the top of the file to the bottom.
func (instrumentor *Instrumentor) currentPath() string {
	if len(instrumentor.currPos) == 0 {
		return ""
	}
	return strings.Join(instrumentor.currPos, "|")
}

// Stash the original line number associated with this particular node. We do that by mapping a unique
// node identifier -- where it is in the AST, per our deterministic depth-first search -- to the line
// number of that node in the original version of the file.
func (instrumentor *Instrumentor) collectLine(node ast.Node) {
	path := instrumentor.currentPath()
	if len(path) == 0 {
		return
	}
	// The first Ident will pretty much always be the package name. Don't add a //line directive
	// since we'll likely get that in the wrong place, due to the "Package" object being
	// disconnected in the AST from the package name.
	if path == "*ast.File@0|*ast.Ident@0" {
		if instrumentor.logWriter.VerboseLevel(2) {
			instrumentor.logWriter.Printf("Skipping package name path %s for node (%T:%v)", path, node, node)
		}
		return
	}
	// For certain types of nodes we will not create line directives, therefore we can
	// just not collect the lines associated with those nodes.
	switch n := node.(type) {
	case *ast.File:
		return
	case *ast.CommentGroup, *ast.Comment:
		return
	case *ast.GenDecl:
		if n.Tok == token.IMPORT {
			// Don't annotate import statements
			return
		}
	}
	// Get where this node is in the original version of the file.
	p := instrumentor.fset.Position(node.Pos())
	// Map the node to the original line number.
	if instrumentor.logWriter.VerboseLevel(3) {
		instrumentor.logWriter.Printf("collectLine(%T:%v:%s) => %d", node, node, path, p.Line)
	}
	instrumentor.nodeLines[path] = p.Line
}

// Given our current position in the AST, on which line number was this node located
// in the original version of the file? If we don't know (e.g., path is empty, or we
// didn't record the position for whatever reason) return -1.
func (instrumentor *Instrumentor) getOriginalLine() int {
	path := instrumentor.currentPath()
	if len(path) == 0 {
		return -1
	}
	if line, ok := instrumentor.nodeLines[path]; ok {
		return line
	}
	return -1
}

func (instrumentor *Instrumentor) VisitAndInstrument(node ast.Node) ast.Visitor {
	if node == nil {
		instrumentor.popType()
		top, _ := instrumentor.nodeStack.Pop()
		if decl, isDecl := top.(*ast.FuncDecl); isDecl {
			if instrumentor.logWriter.VerboseLevel(2) {
				instrumentor.logWriter.Printf("AddCallbacks Popping function %s", decl.Name.Name)
			}
			instrumentor.funcStack.Pop()
		} else {
			if instrumentor.logWriter.VerboseLevel(2) {
				instrumentor.logWriter.Printf("AddCallbacks Popping node: %v (%T)", top, top)
			}
		}
		return instrumentor
	}

	if isInstrumentationCallback(node) {
		// It is possible for us to start traversing nodes that we've inserted ahead of ourselves,
		// so skip over those since we're not going to instrument the instrumentation, AND it will
		// throw off our accounting needed for collecting line numbers.
		return nil
	}

	instrumentor.pushType(node)
	instrumentor.collectLine(node)

	switch n := node.(type) {
	case *ast.FuncDecl:
		if n.Name.String() == "init" {
			// Don't instrument init functions.
			// They run regardless of what we do, so it is just noise.
			instrumentor.popType()
			return nil
		}
	case *ast.GenDecl:
		if n.Tok != token.VAR {
			instrumentor.popType()
			return nil // constants and types are not interesting
		}

	case *ast.BlockStmt:
		// If it's a switch or select, the body is a list of case clauses; don't tag the block itself.
		if len(n.List) > 0 {
			switch n.List[0].(type) {
			case *ast.CaseClause: // switch
				for _, n := range n.List {
					clause := n.(*ast.CaseClause)
					clause.Body = instrumentor.instrumentEdge(clause.Pos(), clause.End(), clause.Body, false)
				}
				return instrumentor
			case *ast.CommClause: // select
				for _, n := range n.List {
					clause := n.(*ast.CommClause)
					clause.Body = instrumentor.instrumentEdge(clause.Pos(), clause.End(), clause.Body, false)
				}
				return instrumentor
			}
		}
		n.List = instrumentor.instrumentEdge(n.Lbrace, n.Rbrace+1, n.List, true) // +1 to step past closing brace.
	case *ast.IfStmt:
		if n.Init != nil {
			ast.Walk(instrumentor, n.Init)
		}
		if n.Cond != nil {
			ast.Walk(instrumentor, n.Cond)
		}
		ast.Walk(instrumentor, n.Body)
		if n.Else == nil {
			// Add else because we want coverage for "not taken".
			n.Else = &ast.BlockStmt{
				Lbrace: n.Body.End(),
				Rbrace: n.Body.End(),
			}
		}
		switch stmt := n.Else.(type) {
		case *ast.IfStmt:
			block := &ast.BlockStmt{
				Lbrace: n.Body.End(), // Start at end of the "if" block so the covered part looks like it starts at the "else".
				List:   []ast.Stmt{stmt},
				Rbrace: stmt.End(),
			}
			n.Else = block
		case *ast.BlockStmt:
			stmt.Lbrace = n.Body.End() // Start at end of the "if" block so the covered part looks like it starts at the "else".
		default:
			instrumentor.logWriter.Fatalf("Unexpected node type in if : %v (%T)", n, n)
		}
		ast.Walk(instrumentor, n.Else)
		instrumentor.popType()
		return nil
	case *ast.ForStmt:
		// TODO: handle increment statement
	case *ast.SelectStmt:
		// Don't annotate an empty select - creates a syntax error.
		if n.Body == nil || len(n.Body.List) == 0 {
			instrumentor.popType()
			return nil
		}
	case *ast.SwitchStmt:
		hasDefault := false
		if n.Body == nil {
			n.Body = new(ast.BlockStmt)
		}
		for _, s := range n.Body.List {
			if cas, ok := s.(*ast.CaseClause); ok && cas.List == nil {
				hasDefault = true
				break
			}
		}
		if !hasDefault {
			// Add default case to get additional coverage.
			n.Body.List = append(n.Body.List, &ast.CaseClause{})
		}

		// Don't annotate an empty switch - creates a syntax error.
		if n.Body == nil || len(n.Body.List) == 0 {
			instrumentor.popType()
			return nil
		}
	case *ast.TypeSwitchStmt:
		// Don't annotate an empty type switch - creates a syntax error.
		// TODO: add default case
		if n.Body == nil || len(n.Body.List) == 0 {
			instrumentor.popType()
			return nil
		}
	case *ast.BinaryExpr:
		if n.Op == token.LAND || n.Op == token.LOR {
			// Expand the right expression to a comparison with the intrinsic "true". Copy its position to these new nodes.
			compareYToTrue := &ast.BinaryExpr{X: n.Y, OpPos: n.Y.End(), Op: token.EQL, Y: ast.NewIdent("true")}
			// Wrap this comparison in a closure.
			closureWithInstrumentation := &ast.FuncLit{
				Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("bool")}}}},
				Body: &ast.BlockStmt{Lbrace: n.Y.End(), List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{compareYToTrue}}}, Rbrace: n.OpPos},
			}
			closureCallExpression := &ast.CallExpr{
				Lparen: n.Y.End(),
				Fun:    closureWithInstrumentation,
				Rparen: n.Y.End(),
			}
			// We have seen cases in which the value of this logical expression cannot be passed to a function
			// that takes a specialized Boolean type, and so the instrumented code cannot be compiled. This
			// comparison to "true" gets us around this oddity of the Go type system (or a bug in the compiler).
			compareClosureInvocationToTrue := &ast.BinaryExpr{X: closureCallExpression, OpPos: n.Y.End(), Op: token.EQL, Y: ast.NewIdent("true")}
			n.Y = compareClosureInvocationToTrue
		}
	case *ast.BadExpr:
		instrumentor.logWriter.Fatalf("Invalid input: %v (%T)", node, node)
	case *ast.BadDecl:
		instrumentor.logWriter.Fatalf("Invalid input: %v (%T)", node, node)
	}
	// If nil is returned, the children of the current node will not be visited. Now push the node so we can pop it later.
	if decl, isDecl := node.(*ast.FuncDecl); isDecl {
		if instrumentor.logWriter.VerboseLevel(2) {
			instrumentor.logWriter.Printf("AddCallbacks Entering function %s", decl.Name.Name)
		}
		instrumentor.funcStack.Push(node)
	} else {
		if instrumentor.logWriter.VerboseLevel(2) {
			instrumentor.logWriter.Printf("AddCallbacks Pushing node: %v (%T)", node, node)
		}
	}
	instrumentor.nodeStack.Push(node)
	return instrumentor
}

func (instrumentor *Instrumentor) VisitAndAddLines(node ast.Node) ast.Visitor {
	if node == nil {
		instrumentor.popType()
		top, _ := instrumentor.nodeStack.Pop()
		if decl, isDecl := top.(*ast.FuncDecl); isDecl {
			if instrumentor.logWriter.VerboseLevel(2) {
				instrumentor.logWriter.Printf("AddLines Popping function %s", decl.Name.Name)
			}
			instrumentor.funcStack.Pop()
		} else {
			if instrumentor.logWriter.VerboseLevel(2) {
				instrumentor.logWriter.Printf("AddLines Popping node: %v (%T)", top, top)
			}
		}
		return instrumentor
	}

	if isInstrumentationCallback(node) {
		return nil
	}
	instrumentor.pushType(node)
	lineNum := instrumentor.getOriginalLine()
	if lineNum > 0 {
		comment := instrumentor.createLineDirective(lineNum, &node)
		if comment != nil {
			if instrumentor.logWriter.VerboseLevel(3) {
				instrumentor.logWriter.Printf("Created line directive for %s line %d", instrumentor.currentPath(), lineNum)
			}
			instrumentor.comments = append(instrumentor.comments, comment)
		} else {
			if instrumentor.logWriter.VerboseLevel(3) {
				instrumentor.logWriter.Printf("Not creating line directive for line %d path %s", lineNum, instrumentor.currentPath())
			}
		}
	} else {
		if instrumentor.logWriter.VerboseLevel(3) {
			instrumentor.logWriter.Printf("No line number available for %v=%v", node, instrumentor.currentPath())
		}
	}

	switch n := node.(type) {
	case *ast.ExprStmt:
	case *ast.Ident:
	case *ast.FuncDecl:
		if n.Name.String() == "init" {
			instrumentor.popType()
			return nil
		}
	case *ast.GenDecl:
		if n.Tok != token.VAR {
			instrumentor.popType()
			return nil // constants and types have nothing under them not interesting
		}
	case *ast.IfStmt:
		if n.Init != nil {
			ast.Walk(instrumentor, n.Init)
		}
		if n.Cond != nil {
			ast.Walk(instrumentor, n.Cond)
		}
		ast.Walk(instrumentor, n.Body)
		if n.Else != nil {
			ast.Walk(instrumentor, n.Else)
		}
		instrumentor.popType()
		return nil
	case *ast.SelectStmt:
		// Don't visit an empty select
		if n.Body == nil || len(n.Body.List) == 0 {
			instrumentor.popType()
			return nil
		}
	case *ast.SwitchStmt:
		// Don't visit an empty switch
		if n.Body == nil || len(n.Body.List) == 0 {
			instrumentor.popType()
			return nil
		}
	case *ast.TypeSwitchStmt:
		// Don't visit an empty type switch
		if n.Body == nil || len(n.Body.List) == 0 {
			instrumentor.popType()
			return nil
		}
	}
	// If nil is returned, the children of the current node will not be visited. Now push the node so we can pop it later.
	if decl, isDecl := node.(*ast.FuncDecl); isDecl {
		if instrumentor.logWriter.VerboseLevel(2) {
			instrumentor.logWriter.Printf("AddLines Entering function %s", decl.Name.Name)
		}
		instrumentor.funcStack.Push(node)
	} else {
		if instrumentor.logWriter.VerboseLevel(2) {
			instrumentor.logWriter.Printf("AddLines Pushing node: %v (%T)", node, node)
		}
	}
	instrumentor.nodeStack.Push(node)
	return instrumentor
}

// Visit is part of the FileWalker interface.
// TODO: (justin.moore) See how difficult it would be to merge the Visit sub-functions back into a
// single Visit() function, and just switch on control flow based on the addLines boolean, rather
// than duplicating most of the switch statement in each function.
func (instrumentor *Instrumentor) Visit(node ast.Node) ast.Visitor {
	if instrumentor.logWriter.VerboseLevel(2) {
		instrumentor.logWriter.Printf("Visit(%v, %T:%v => %T:%v)", instrumentor.addLines, &node, &node, node, node)
	}
	if instrumentor.addLines {
		return instrumentor.VisitAndAddLines(node)
	} else {
		return instrumentor.VisitAndInstrument(node)
	}
}

func (instrumentor *Instrumentor) statementBoundary(s ast.Stmt) token.Pos {
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

func (instrumentor *Instrumentor) instrumentEdge(pos, blockEnd token.Pos, list []ast.Stmt, extendToClosingBrace bool) []ast.Stmt {
	// Special case: make sure we add a counter to an empty block. Can't do this below
	// or we will add a counter to an empty statement list after, say, a return statement.
	if len(list) == 0 {
		return []ast.Stmt{instrumentor.newEdge(pos, blockEnd)}
	}
	// We have a block (statement list), but it may have several basic blocks due to the
	// appearance of statements that affect the flow of control.
	var newList []ast.Stmt
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
			newList = append(newList, instrumentor.newEdge(pos, end))
		}
		newList = append(newList, list[0:last]...)
		list = list[last:]
		if len(list) == 0 {
			break
		}
		pos = list[0].Pos()
	}
	return newList
}

func (instrumentor *Instrumentor) endsBasicSourceBlock(s ast.Stmt) bool {
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

func (instrumentor *Instrumentor) newEdge(start, end token.Pos) ast.Stmt {
	instrumentor.CurrentEdge++

	s := instrumentor.fset.Position(start)
	e := instrumentor.fset.Position(end)
	maybe_decl, _ := instrumentor.funcStack.Peek()
	decl, isDecl := maybe_decl.(*ast.FuncDecl)
	fname := ""
	if isDecl {
		fname = decl.Name.Name
	}

	err := instrumentor.SymbolTable.WritePosition(SymbolTablePosition{
		Path:        instrumentor.fullName,
		Function:    fname,
		StartLine:   s.Line,
		StartColumn: s.Column,
		EndLine:     e.Line,
		EndColumn:   e.Column,
		Edge:        instrumentor.CurrentEdge,
	})
	if err != nil {
		instrumentor.logWriter.Fatalf("Could not write symbol table line: %s", err.Error())
	}

	idx := &ast.BasicLit{
		Kind:  token.INT,
		Value: strconv.Itoa(instrumentor.CurrentEdge),
	}
	caller := &ast.SelectorExpr{
		X:   ast.NewIdent(InstrumentationPackageAlias),
		Sel: ast.NewIdent(AntithesisCallbackFunction),
	}
	return &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun:  caller,
			Args: []ast.Expr{idx},
		},
	}
}

// In the AST we'll see a SelectExpr which will have:
// - a package equal to InstrumentationPackageAlias
// - a function name equal to AntithesisCallbackFunction
//
// After those the next thing we see should be an integer literal. I.e.,
//
// __antithesis_instrumentation__.Notify(6)
// |       Package name          |  Fn  |^~ Integer literal
//
// In the AST that will be: *ast.SelectorExpr:&{__antithesis_instrumentation__ Notify})
// - Selector{X=(*ast.Ident:__antithesis_instrumentation__), Sel=(*ast.Ident:Notify)}
// Followed by: *ast.BasicLit:&{18366 INT 1})
// - BasicLit{Kind=(token.Token:INT), Value=(string:1)}
func isInstrumentationCallback(n ast.Node) bool {
	node, isExp := n.(*ast.ExprStmt)
	if !isExp {
		return false
	}
	if node.X == nil {
		return false
	}
	call, callOk := node.X.(*ast.CallExpr)
	if !callOk {
		return false
	}
	if call.Fun == nil || call.Args == nil {
		return false
	}
	sel, selOk := call.Fun.(*ast.SelectorExpr)
	if !selOk {
		return false
	}
	if sel.X == nil || sel.Sel == nil {
		return false
	}
	x, xOk := sel.X.(*ast.Ident)
	if !xOk {
		return false
	}
	return x.Name == InstrumentationPackageAlias && sel.Sel.Name == AntithesisCallbackFunction
}

func (instrumentor *Instrumentor) formatInstrumentedAst(inputPath string, astFile *ast.File, addShim bool) (string, error) {
	if addShim {
		astutil.AddNamedImport(instrumentor.fset, astFile, InstrumentationPackageAlias, instrumentor.ShimPkg)
	}
	writer := strings.Builder{}
	formatError := format.Node(&writer, instrumentor.fset, astFile)
	if formatError != nil {
		instrumentor.logWriter.Printf("Error: Could not write instrumented AST from %s: %v", inputPath, formatError)
		return "", formatError
	}

	source := writer.String()
	if _, parseError := parser.ParseFile(&token.FileSet{}, inputPath, source, parser.ParseComments); parseError != nil {
		instrumentor.logWriter.Printf("Error: Instrumented source for %s could not be parsed; simply copying original: %s", inputPath, parseError)
		return "", parseError
	}

	return source, nil
}

func sortComments(comments []*ast.CommentGroup) {
	sort.Slice(comments, func(i, j int) bool {
		// Get the position of the first comment in each.
		// Based on the documentation of the Golang AST package, we can assume that
		// the List element will have non-zero length:
		// https://pkg.go.dev/go/ast#CommentGroup
		iFirst := comments[i].List[0]
		jFirst := comments[j].List[0]
		// The Slash member is the position of "/" starting the comment:
		// https://pkg.go.dev/go/ast#Comment
		return iFirst.Slash < jFirst.Slash
	})
}
