package assertions

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/antithesishq/antithesis-sdk-go/assert"
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

type AntGuidance struct {
	*GuidanceFuncInfo
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
	guidanceHintMap    GuidanceHints
	fset               *token.FileSet
	logWriter          *common.LogWriter
	symbolTableName    string
	funcName           string
	receiver           string
	packageName        string
	moduleName         string
	notifierPackage    string
	notifierModuleName string
	baseInputDir       string
	baseTargetDir      string
	expects            []*AntExpect
	guidance           []*AntGuidance
	imports            []string
	filesCataloged     int
	verbose            bool
}

// filter Guidance to just numeric
func (aScanner *AssertionScanner) numericGuidance() []*AntGuidance {
	numeric_guidance := []*AntGuidance{}
	for _, aG := range aScanner.guidance {
		gp := aG.GuidanceFn
		if gp == assert.GuidanceFnMaximize || gp == assert.GuidanceFnMinimize {
			numeric_guidance = append(numeric_guidance, aG)
		}
	}
	return numeric_guidance
}

// filter Guidance to just boolean
func (aScanner *AssertionScanner) booleanGuidance() []*AntGuidance {
	boolean_guidance := []*AntGuidance{}
	for _, aG := range aScanner.guidance {
		gp := aG.GuidanceFn
		if gp == assert.GuidanceFnWantAll || gp == assert.GuidanceFnWantNone {
			boolean_guidance = append(boolean_guidance, aG)
		}
	}
	return boolean_guidance
}

func NewAssertionScanner(verbose bool, moduleName string, symbolTableName string, sourceDir string, targetDir string) *AssertionScanner {
	logWriter := common.GetLogWriter()
	if logWriter.VerboseLevel(2) {
		logWriter.Printf(">> Module: %s\n", moduleName)
	}

	aScanner := AssertionScanner{
		moduleName:       moduleName,
		fset:             token.NewFileSet(),
		imports:          []string{},
		expects:          []*AntExpect{},
		guidance:         []*AntGuidance{},
		verbose:          verbose,
		funcName:         "",
		receiver:         "",
		packageName:      "",
		baseInputDir:     sourceDir,
		baseTargetDir:    targetDir,
		assertionHintMap: SetupHintMap(),
		guidanceHintMap:  SetupGuidanceHintMap(),
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

	expects := aScanner.expects
	has_expects := len(expects) > 0

	numeric_guidance := aScanner.numericGuidance()
	has_numeric_guidance := len(numeric_guidance) > 0

	boolean_guidance := aScanner.booleanGuidance()
	has_boolean_guidance := len(boolean_guidance) > 0

	genInfo := GenInfo{
		ExpectedVals:        expects,
		NumericGuidanceVals: numeric_guidance,
		BooleanGuidanceVals: boolean_guidance,
		AssertPackageName:   common.AssertPackageName(),
		VersionText:         versionText,
		CreateDate:          createDate,
		HasAssertions:       has_expects,
		HasNumericGuidance:  has_numeric_guidance,
		HasBooleanGuidance:  has_boolean_guidance,
		ConstMap:            aScanner.getConstMap(),
		logWriter:           common.GetLogWriter(),
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
	aScanner.packageName = ""
	aScanner.receiver = ""
}

func (aScanner *AssertionScanner) module_relative_name(file_path string) string {
	base_dir := common.CanonicalizeDirectory(aScanner.baseInputDir)
	full_file_path := common.CanonicalizeDirectory(file_path)

	// skip over the base inputDirectory from the inputfilename,
	// and create the output directories needed
	if !strings.HasPrefix(full_file_path, base_dir) {
		return file_path
	}

	skipLength := len(base_dir)
	if len(base_dir) > 1 {
		skipLength += 1
	}
	revised_path := full_file_path[skipLength:]
	return revised_path
}

func (aScanner *AssertionScanner) node_inspector(x ast.Node) bool {
	var call_expr *ast.CallExpr
	var func_decl *ast.FuncDecl
	var package_file *ast.File
	var import_spec *ast.ImportSpec
	var fun_expr ast.Expr
	var call_args []ast.Expr
	var ok bool
	var path_name string

	assertPackageName := common.AssertPackageName()

	if aScanner.packageName == "" {
		if package_file, ok = x.(*ast.File); ok {
			// subtract aScanner.baseInputDir from the full file name in full_position
			// and use that top qualify packageName
			// eg. the "abc" package within "ms-test" module is imported like this:
			//    import "ms-test/abc"
			//
			// The abc package has files named f1.go and f2.go
			// when instrumenting file f1.go, it is located in "ms-test/abc/f1.go"
			// similarly, instrumenting file f2.go, it is located in "ms-test/abc/f2.go"
			// Note that the package_file.Name.Name is simply "abc" corresponding
			// to the "package abc" statement at the top of f1.go (or f2.go)
			//
			// The package name that should be placed in aScanner.packageName is "ms-test/abc"
			// and should not simply be "abc".
			//
			// Go programs always, by definition, all contain a package named "main".  Files
			// that are instrumented from package "main" should set aScanner.packageName to
			// "main", and not to "ms-test/main".
			//

			cooked_moduleName := aScanner.moduleName // source filename
			base_target_dir := aScanner.baseTargetDir
			if strings.HasPrefix(cooked_moduleName, base_target_dir) {
				lx := len(base_target_dir)
				if len(base_target_dir) > 1 {
					lx += 1
				}
				cooked_moduleName = cooked_moduleName[lx:]
			}
			xpos := aScanner.fset.Position(package_file.Pos())
			relative_name := aScanner.module_relative_name(xpos.Filename)
			idx := strings.LastIndex(relative_name, string(os.PathSeparator))
			if idx == -1 {
				aScanner.packageName = package_file.Name.Name
			} else {
				prefix := relative_name[0:idx]
				aScanner.packageName = fmt.Sprintf("%s%c%s", cooked_moduleName, os.PathSeparator, prefix)
			}
		}
	}

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

		return true // ast.inspect() can deal with this
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
			relative_file_path := aScanner.module_relative_name(full_position.Filename)
			expr_text := analyzed_expr(aScanner.imports, sel_expr.X)
			target_func := sel_expr.Sel.Name
			if func_hints := aScanner.assertionHintMap.HintsForName(target_func); func_hints != nil && expr_text != "" {
				test_name := arg_at_index(call_args, func_hints.MessageArg)
				if test_name == common.NAME_NOT_AVAILABLE {
					generated_msg := fmt.Sprintf("%s[%d]", relative_file_path, full_position.Line)
					test_name = fmt.Sprintf("Message from %s", strconv.Quote(generated_msg))
				}
				expect := AntExpect{
					Assertion:         target_func,
					Message:           test_name,
					Classname:         aScanner.packageName,
					Funcname:          aScanner.funcName,
					Receiver:          aScanner.receiver,
					Filename:          relative_file_path,
					Line:              full_position.Line,
					AssertionFuncInfo: func_hints,
				}
				aScanner.expects = append(aScanner.expects, &expect)
			} // assertionHint

			if guidance_func_hints := aScanner.guidanceHintMap.GuidanceHintsForName(target_func); guidance_func_hints != nil && expr_text != "" {
				test_name := arg_at_index(call_args, guidance_func_hints.MessageArg)
				if test_name == common.NAME_NOT_AVAILABLE {
					generated_msg := fmt.Sprintf("%s[%d]", relative_file_path, full_position.Line)
					test_name = fmt.Sprintf("Message from %s", strconv.Quote(generated_msg))
				}
				// The registration for the Guidance function itself
				guidance_expect := AntGuidance{
					Assertion:        target_func,
					Message:          test_name,
					Classname:        aScanner.packageName,
					Funcname:         aScanner.funcName,
					Receiver:         aScanner.receiver,
					Filename:         relative_file_path,
					Line:             full_position.Line,
					GuidanceFuncInfo: guidance_func_hints,
				}
				aScanner.guidance = append(aScanner.guidance, &guidance_expect)

				// The Related Assertion derived from target_func("AlwaysGreaterThan") => derived_target_func("Always")
				expect := AntExpect{
					Assertion: target_func_from_guidance(target_func),
					Message:   test_name,
					Classname: aScanner.packageName,
					Funcname:  aScanner.funcName,
					Receiver:  aScanner.receiver,
					Filename:  relative_file_path,
					Line:      full_position.Line,
					// NOTE: AssertionFuncInfo.TargetFunc is a guidance func name
					// and AssertionFuncInfo.MessageArg refers to a Guidance Function argument number.
					//
					// GenerateAssertionsCatalog() does not use either of TargetFunc or MessageArg
					// attributes of AssertionFuncInfo, so it is safe to pass the AssertionFuncInfo
					// from the guidance func here.
					AssertionFuncInfo: &guidance_func_hints.AssertionFuncInfo,
				}
				aScanner.expects = append(aScanner.expects, &expect)
			} // assertionHint
		}
	}
	return true
}

func target_func_from_guidance(guidance_func string) string {
	target_func := ""
	if strings.HasPrefix(guidance_func, "Always") {
		target_func = "Always"
	} else if strings.HasPrefix(guidance_func, "Sometimes") {
		target_func = "Sometimes"
	}
	return target_func
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

// --------------------------------------------------------------------------------
// The 'ConstMap' is used by GenerateAssertionsCatalog to define the
// go constants referenced in the registration statements
// that are generated.
//
// This will prevent go build warnings/errors related to defining
// a const that is not actually used anywhere.
//
// Example, if none of the generated registrations use an AssertType
// "reachability", then the corresponding go const should not be
// output to the generated Assertions Catalog '.go' file.
// --------------------------------------------------------------------------------
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
