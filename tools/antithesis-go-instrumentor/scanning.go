package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path"
	"strconv"
)

type AntExpect struct {
	Assertion string
	Message   string
	Classname string
	Funcname  string
	Receiver  string
	Filename  string
	Line      int
	*AssertionFuncInfo
}

type AssertionScanningInfo struct {
	module_name        string
	fset               *token.FileSet
	imports            []string
	expects            []*AntExpect
	verbose            bool
	func_name          string
	receiver           string
	package_name       string
	assertion_hint_map AssertionHints // = setup_hint_map()
	symbol_table_name  string
}

const NAME_NOT_AVAILABLE = "anonymous"
const ANTITHESIS_SDK_PACKAGE = "github.com/antithesishq/antithesis-sdk-go/assert"
const INSTRUMENTATION_PACKAGE_NAME = "github.com/antithesishq/antithesis-sdk-go/instrumentation"

func NewScanningInfo(verbose bool, module_name string, symbol_table_name string) *AssertionScanningInfo {
	if verbose_level(2) {
		logger.Printf(">> Module: %s\n", module_name)
	}
	aSI := AssertionScanningInfo{
		module_name:        module_name,
		fset:               token.NewFileSet(),
		imports:            []string{},
		expects:            []*AntExpect{},
		verbose:            verbose,
		func_name:          "",
		receiver:           "",
		package_name:       "",
		assertion_hint_map: setup_hint_map(),
		symbol_table_name:  symbol_table_name,
	}
	return &aSI
}

func (aSI *AssertionScanningInfo) reset_for_package(packagename string, package_path string) {
	if verbose_level(2) {
		logger.Printf(">>   Package: %s (%s)\n", packagename, package_path)
	}
	aSI.package_name = packagename
}

func (aSI *AssertionScanningInfo) reset_for_file(file_path string) {
	if verbose_level(2) {
		logger.Printf(">>     File: %s\n", file_path)
	}
	aSI.imports = []string{}
	aSI.func_name = ""
	aSI.receiver = ""
}

func (aSI *AssertionScanningInfo) ScanFile(file_path string) {
	var file *ast.File
	var err error

	aSI.reset_for_file(file_path)
	if file, err = parser.ParseFile(aSI.fset, file_path, nil, 0); err != nil {
		panic(err)
	}

	ast.Inspect(file, aSI.node_inspector)
}

func (aSI *AssertionScanningInfo) node_inspector(x ast.Node) bool {
	var call_expr *ast.CallExpr
	var func_decl *ast.FuncDecl
	var import_spec *ast.ImportSpec
	var fun_expr ast.Expr
	var call_args []ast.Expr
	var ok bool
	var path_name string

	if import_spec, ok = x.(*ast.ImportSpec); ok {
		path_name, _ = strconv.Unquote(import_spec.Path.Value)
		alias := ""
		if import_spec.Name != nil {
			alias = import_spec.Name.Name
		}
		if path_name == ANTITHESIS_SDK_PACKAGE {
			call_qualifier := path.Base(path_name)
			if alias != "" {
				call_qualifier = alias
			}
			aSI.imports = append(aSI.imports, call_qualifier)
		}

		return true // you deal with this
	}

	// Track current func_name and receiver (type)
	if func_decl, ok = x.(*ast.FuncDecl); ok {
		aSI.func_name = NAME_NOT_AVAILABLE
		if func_ident := func_decl.Name; func_ident != nil {
			aSI.func_name = func_ident.Name
		}
		aSI.receiver = ""
		if recv := func_decl.Recv; recv != nil {
			if num_fields := recv.NumFields(); num_fields > 0 {
				if field_list := recv.List; field_list != nil {
					if recv_type := field_list[0].Type; recv_type != nil {
						aSI.receiver = types.ExprString(recv_type)
					}
				}
			}
		}
		if verbose_level(2) {
			logger.Printf(">>       Func: %s %s\n", aSI.func_name, aSI.receiver)
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
		if _, ok = fun_expr.(*ast.Ident); ok {
			return true // recurse further
		}

		var sel_expr *ast.SelectorExpr
		if sel_expr, ok = fun_expr.(*ast.SelectorExpr); ok {
			full_position := aSI.fset.Position(sel_expr.Pos())
			expr_text := analyzed_expr(aSI.imports, sel_expr.X)
			target_func := sel_expr.Sel.Name
			if func_hints := aSI.assertion_hint_map.hints_for_name(target_func); func_hints != nil && expr_text != "" {
				test_name := arg_at_index(call_args, 0)
				expect := AntExpect{
					Assertion:         target_func,
					Message:           test_name,
					Classname:         aSI.package_name,
					Funcname:          aSI.func_name,
					Receiver:          aSI.receiver,
					Filename:          full_position.Filename,
					Line:              full_position.Line,
					AssertionFuncInfo: func_hints,
				}
				aSI.expects = append(aSI.expects, &expect)
			}
			return false
		}
		return false
	}
	return true
}

func arg_at_index(args []ast.Expr, idx int) string {
	if args == nil || idx < 0 || len(args) <= idx {
		return NAME_NOT_AVAILABLE
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
		return ident.String()
	}
	return NAME_NOT_AVAILABLE
}

func analyzed_expr(imports []string, expr ast.Expr) string {
	var expr_name string = ""
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
	Expecting_true
	Expecting_false
	Universal_test
	Existential_test
	Reachability_test
	Num_conditions
)

func (aSI *AssertionScanningInfo) GetConstMap() map[string]bool {
	cond_tracker := make([]bool, Num_conditions, Num_conditions)
	if len(aSI.expects) > 0 {
		cond_tracker[Not_hit] = true
	}
	for _, an_expect := range aSI.expects {
		pAFI := an_expect.AssertionFuncInfo
		if pAFI.MustHit {
			cond_tracker[Must_be_hit] = true
		} else {
			cond_tracker[Optionally_hit] = true
		}
		if pAFI.Expecting {
			cond_tracker[Expecting_true] = true
		} else {
			cond_tracker[Expecting_false] = true
		}
		if pAFI.Condition {
			cond_tracker[Cond_true] = true
		} else {
			cond_tracker[Cond_false] = true
		}
		if pAFI.AssertType == "every" {
			cond_tracker[Universal_test] = true
		}
		if pAFI.AssertType == "some" {
			cond_tracker[Existential_test] = true
		}
		if pAFI.AssertType == "none" {
			cond_tracker[Reachability_test] = true
		}
	}

	const_map := make(map[string]bool)
	const_map["cond_false"] = cond_tracker[Cond_false]
	const_map["cond_true"] = cond_tracker[Cond_true]
	const_map["was_hit"] = cond_tracker[Was_hit]
	const_map["not_hit"] = cond_tracker[Not_hit]
	const_map["must_be_hit"] = cond_tracker[Must_be_hit]
	const_map["optionally_hit"] = cond_tracker[Optionally_hit]
	const_map["expecting_true"] = cond_tracker[Expecting_true]
	const_map["expecting_false"] = cond_tracker[Expecting_false]
	const_map["universal_test"] = cond_tracker[Universal_test]
	const_map["existential_test"] = cond_tracker[Existential_test]
	const_map["reachability_test"] = cond_tracker[Reachability_test]
	return const_map
}

func (aSI *AssertionScanningInfo) WriteAssertionCatalog(edge_count int) {
	using_symbols := ""
	needs_coverage := false
	if len(aSI.symbol_table_name) > 0 {
		using_symbols = aSI.symbol_table_name
		needs_coverage = true
	}
	gen_info := GenInfo{
		ExpectedVals:               aSI.expects,
		ExpectPackageName:          ANTITHESIS_SDK_PACKAGE,
		InstrumentationPackageName: INSTRUMENTATION_PACKAGE_NAME,
		SymbolTableName:            using_symbols,
		EdgeCount:                  edge_count,
		HasAssertions:              (len(aSI.expects) > 0),
		ConstMap:                   aSI.GetConstMap(),
		NeedsCoverage:              needs_coverage,
	}

	// destination name is expected to be a file_path
	// destination name will have '_antithesis_catalog.go' appended to it
	generate_expects(aSI.module_name, &gen_info)
}
