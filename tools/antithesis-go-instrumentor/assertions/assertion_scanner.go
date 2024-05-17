package assertions

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path"
	"strconv"
	"time"

	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
)

type AntExpect struct {
	*AssertionFuncInfo
	Assertion string
	Message   string
	Classname string
	Funcname  string
	Receiver  string
	Filename  string
	Line      int
}

// Capitalized struct items are accessed outside this file
type AssertionScanner struct {
	assertionHintMap   AssertionHints
	fset               *token.FileSet
	logWriter          *common.LogWriter
	symbolTableName    string
	funcName           string
	receiver           string
	packageName        string
	moduleName         string
	notifierPackage    string
	notifierModuleName string
	expects            []*AntExpect
	imports            []string
	filesCataloged     int
	verbose            bool
}

func NewAssertionScanner(verbose bool, moduleName string, symbolTableName string /*, notifierPackage string*/) *AssertionScanner {
	logWriter := common.GetLogWriter()
	if logWriter.VerboseLevel(2) {
		logWriter.Printf(">> Module: %s\n", moduleName)
	}

	aScanner := AssertionScanner{
		moduleName:       moduleName,
		fset:             token.NewFileSet(),
		imports:          []string{},
		expects:          []*AntExpect{},
		verbose:          verbose,
		funcName:         "",
		receiver:         "",
		packageName:      "",
		assertionHintMap: SetupHintMap(),
		symbolTableName:  symbolTableName,
		filesCataloged:   0,
		logWriter:        logWriter,
	}
	return &aScanner
}

func (aScanner *AssertionScanner) GetLogger() *common.LogWriter {
	return aScanner.logWriter
}

func (aScanner *AssertionScanner) ScanFile(file_path string) {
	var file *ast.File
	var err error

	aScanner.logWriter.Printf("Cataloging %s", file_path)
	aScanner.reset_for_file(file_path)
	if file, err = parser.ParseFile(aScanner.fset, file_path, nil, 0); err != nil {
		panic(err)
	}

	ast.Inspect(file, aScanner.node_inspector)
	aScanner.filesCataloged++
}

func (aScanner *AssertionScanner) HasAssertionsDefined() bool {
	return len(aScanner.expects) > 0
}

func (aScanner *AssertionScanner) WriteAssertionCatalog(versionText string) {
	now := time.Now()
	createDate := now.Format("Mon Jan 2 15:04:05 MST 2006")

	genInfo := GenInfo{
		ExpectedVals:      aScanner.expects,
		AssertPackageName: common.AssertPackageName(),
		VersionText:       versionText,
		CreateDate:        createDate,
		HasAssertions:     (len(aScanner.expects) > 0),
		ConstMap:          aScanner.getConstMap(),
		logWriter:         common.GetLogWriter(),
	}

	// destination name is expected to be a file_path
	// destination name will have '_antithesis_catalog.go' appended to it
	GenerateAssertionsCatalog(aScanner.moduleName, &genInfo)
}

func (aScanner *AssertionScanner) SummarizeWork() {
	numCataloged := aScanner.filesCataloged
	aScanner.logWriter.Printf("%d '.go' %s cataloged", numCataloged, common.Pluralize(numCataloged, "file"))
}

func (aScanner *AssertionScanner) reset_for_file(file_path string) {
	if aScanner.logWriter.VerboseLevel(2) {
		aScanner.logWriter.Printf(">>     File: %s\n", file_path)
	}
	aScanner.imports = []string{}
	aScanner.funcName = ""
	aScanner.receiver = ""
}

func (aScanner *AssertionScanner) node_inspector(x ast.Node) bool {
	var call_expr *ast.CallExpr
	var func_decl *ast.FuncDecl
	var import_spec *ast.ImportSpec
	var fun_expr ast.Expr
	var call_args []ast.Expr
	var ok bool
	var path_name string

	assertPackageName := common.AssertPackageName()

	if import_spec, ok = x.(*ast.ImportSpec); ok {
		path_name, _ = strconv.Unquote(import_spec.Path.Value)
		alias := ""
		if import_spec.Name != nil {
			alias = import_spec.Name.Name
		}
		if path_name == assertPackageName {
			call_qualifier := path.Base(path_name)
			if alias != "" {
				call_qualifier = alias
			}
			aScanner.imports = append(aScanner.imports, call_qualifier)
		}

		return true // you deal with this
	}

	// Track current funcName and receiver (type)
	if func_decl, ok = x.(*ast.FuncDecl); ok {
		aScanner.funcName = common.NAME_NOT_AVAILABLE
		if func_ident := func_decl.Name; func_ident != nil {
			aScanner.funcName = func_ident.Name
		}
		aScanner.receiver = ""
		if recv := func_decl.Recv; recv != nil {
			if num_fields := recv.NumFields(); num_fields > 0 {
				if field_list := recv.List; field_list != nil {
					if recv_type := field_list[0].Type; recv_type != nil {
						aScanner.receiver = types.ExprString(recv_type)
					}
				}
			}
		}
		if aScanner.logWriter.VerboseLevel(2) {
			aScanner.logWriter.Printf(">>       Func: %s %s\n", aScanner.funcName, aScanner.receiver)
		}
	}

	if call_expr, ok = x.(*ast.CallExpr); ok {
		fun_expr = call_expr.Fun
		call_args = call_expr.Args

		// TODO Check the behavior when 'dot-import' is used to import
		// a package directly into a source file's namespace
		//
		// All supported use cases are expected to be identified by
		// ast.SelectorExpr (which specifies an Expression 'X' and a 'Name')
		// For example, the SelectorExpr for strings.HasPrefix()
		// sel_expr.X is "strings"
		// sel_expr.Name is "HasPrefix"
		var call_ident *ast.Ident
		if call_ident, ok = fun_expr.(*ast.Ident); ok {
			call_name := "<anon>"
			if call_ident != nil {
				call_name = call_ident.Name
			}
			if aScanner.logWriter.VerboseLevel(2) {
				aScanner.logWriter.Printf("Found call to %s()\n", call_name)
			}
		}

		var sel_expr *ast.SelectorExpr
		if sel_expr, ok = fun_expr.(*ast.SelectorExpr); ok {
			full_position := aScanner.fset.Position(sel_expr.Pos())
			expr_text := analyzed_expr(aScanner.imports, sel_expr.X)
			target_func := sel_expr.Sel.Name
			if func_hints := aScanner.assertionHintMap.HintsForName(target_func); func_hints != nil && expr_text != "" {
				test_name := arg_at_index(call_args, func_hints.MessageArg)
				if test_name == common.NAME_NOT_AVAILABLE {
					generated_msg := fmt.Sprintf("%s[%d]", full_position.Filename, full_position.Line)
					test_name = fmt.Sprintf("Message from %s", strconv.Quote(generated_msg))
				}
				expect := AntExpect{
					Assertion:         target_func,
					Message:           test_name,
					Classname:         aScanner.packageName,
					Funcname:          aScanner.funcName,
					Receiver:          aScanner.receiver,
					Filename:          full_position.Filename,
					Line:              full_position.Line,
					AssertionFuncInfo: func_hints,
				}
				aScanner.expects = append(aScanner.expects, &expect)
			}
		}
	}
	return true
}

func arg_at_index(args []ast.Expr, idx int) string {
	if args == nil || idx < 0 || len(args) <= idx {
		return common.NAME_NOT_AVAILABLE
	}
	arg := args[idx]

	var basic_lit *ast.BasicLit
	var basic_lit2 *ast.BasicLit
	var ident *ast.Ident
	var value_spec *ast.ValueSpec
	var ok bool

	// A string literal was provided - nice
	if basic_lit, ok = arg.(*ast.BasicLit); ok {
		text, _ := strconv.Unquote(basic_lit.Value)
		return text
	}

	// Not so nice.
	// A reference to a const or a var or an indexed value was provided
	//
	// Dig in and see if is resolvable at compile-time
	// When a const is declared in another file, it might not be available here
	if ident, ok = arg.(*ast.Ident); ok {
		if ident.Obj == nil || ident.Obj.Decl == nil {
			return ident.String()
		}
		if value_spec, ok = ident.Obj.Decl.(*ast.ValueSpec); ok {
			values := value_spec.Values
			if len(values) > 0 {
				this_value := values[0]
				if basic_lit2, ok = this_value.(*ast.BasicLit); ok {
					const_text, _ := strconv.Unquote(basic_lit2.Value)
					return const_text
				}
			}
		}
	}
	return common.NAME_NOT_AVAILABLE
}

func analyzed_expr(imports []string, expr ast.Expr) string {
	expr_name := ""
	if expr_id, ok := expr.(*ast.Ident); ok {
		expr_name = expr_id.Name
	}
	for _, import_name := range imports {
		if import_name == expr_name {
			return expr_name
		}
	}
	return ""
}

const (
	Cond_false = iota
	Cond_true
	Was_hit
	Not_hit
	Must_be_hit
	Optionally_hit
	Universal_test
	Existential_test
	Reachability_test
	Num_conditions
)

func (aScanner *AssertionScanner) getConstMap() map[string]bool {
	cond_tracker := make([]bool, Num_conditions)
	if len(aScanner.expects) > 0 {
		cond_tracker[Not_hit] = true
	}
	for _, an_expect := range aScanner.expects {
		pAFI := an_expect.AssertionFuncInfo
		if pAFI.MustHit {
			cond_tracker[Must_be_hit] = true
		} else {
			cond_tracker[Optionally_hit] = true
		}
		if pAFI.Condition {
			cond_tracker[Cond_true] = true
		} else {
			cond_tracker[Cond_false] = true
		}
		if pAFI.AssertType == "always" {
			cond_tracker[Universal_test] = true
		}
		if pAFI.AssertType == "sometimes" {
			cond_tracker[Existential_test] = true
		}
		if pAFI.AssertType == "reachability" {
			cond_tracker[Reachability_test] = true
		}
	}

	const_map := make(map[string]bool)
	const_map["condFalse"] = cond_tracker[Cond_false]
	const_map["condTrue"] = cond_tracker[Cond_true]
	const_map["wasHit"] = cond_tracker[Was_hit]
	const_map["notHit"] = cond_tracker[Not_hit]
	const_map["mustBeHit"] = cond_tracker[Must_be_hit]
	const_map["optionallyHit"] = cond_tracker[Optionally_hit]
	const_map["universalTest"] = cond_tracker[Universal_test]
	const_map["existentialTest"] = cond_tracker[Existential_test]
	const_map["reachabilityTest"] = cond_tracker[Reachability_test]
	return const_map
}
