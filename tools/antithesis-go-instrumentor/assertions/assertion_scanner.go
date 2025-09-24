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

type WrapperInfo struct {
	Assertion  string
	MessageArg int
}

// Capitalized struct items are accessed outside this file
type AssertionScanner struct {
	assertionHintMap               AssertionHints
	innerToNormalAssertNames       map[string]string
	guidanceHintMap                GuidanceHints
	fset                           *token.FileSet
	logWriter                      *common.LogWriter
	symbolTableName                string
	funcName                       string
	funcDecl                       *ast.FuncDecl
	receiver                       string
	packageName                    string
	shortPackageName               string
	moduleName                     string
	notifierPackage                string
	notifierModuleName             string
	baseInputDir                   string
	baseTargetDir                  string
	expects                        []*AntExpect
	guidance                       []*AntGuidance
	assertPackageImportNames       []string
	importNameToPackageName        map[string]string
	packageNameToWrapperNameToInfo map[string]map[string]WrapperInfo
	filesCataloged                 int
	verbose                        bool
}

// filter Guidance to just numeric
func (aScanner *AssertionScanner) numericGuidance() []*AntGuidance {
	numeric_guidance := []*AntGuidance{}
	for _, aG := range aScanner.guidance {
		gp := aG.GuidanceFn
		if gp == GuidanceFnMaximize || gp == GuidanceFnMinimize {
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
		if gp == GuidanceFnWantAll || gp == GuidanceFnWantNone {
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
		moduleName:                     moduleName,
		fset:                           token.NewFileSet(),
		assertPackageImportNames:       []string{},
		importNameToPackageName:        map[string]string{},
		expects:                        []*AntExpect{},
		guidance:                       []*AntGuidance{},
		verbose:                        verbose,
		funcName:                       "",
		receiver:                       "",
		packageName:                    "",
		baseInputDir:                   sourceDir,
		baseTargetDir:                  targetDir,
		assertionHintMap:               SetupHintMap(),
		innerToNormalAssertNames:       SetupInnerAssertionMap(),
		packageNameToWrapperNameToInfo: map[string]map[string]WrapperInfo{},
		guidanceHintMap:                SetupGuidanceHintMap(),
		symbolTableName:                symbolTableName,
		filesCataloged:                 0,
		logWriter:                      logWriter,
	}
	return &aScanner
}

func (aScanner *AssertionScanner) GetLogger() *common.LogWriter {
	return aScanner.logWriter
}

func (aScanner *AssertionScanner) ScanFileForInnerAssertions(file_path string) {
	var file *ast.File
	var err error

	aScanner.reset_for_file(file_path)
	if file, err = parser.ParseFile(aScanner.fset, file_path, nil, 0); err != nil {
		panic(err)
	}

	aScanner.logWriter.Printf("Catalog preprocessing: looking for assertion wrappers in %s", file_path)
	ast.Inspect(file, aScanner.node_preprocessor)
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
	aScanner.assertPackageImportNames = []string{}
	aScanner.importNameToPackageName = make(map[string]string)
	aScanner.funcName = ""
	aScanner.funcDecl = nil
	aScanner.packageName = ""
	aScanner.shortPackageName = ""
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

func (aScanner *AssertionScanner) setCurrentPackageName(x ast.Node) {
	var package_file *ast.File
	var ok bool

	if aScanner.packageName == "" {
		if package_file, ok = x.(*ast.File); ok {
			// subtract aScanner.baseInputDir from the full file name in full_position
			// and use that to qualify packageName
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

			// Also set the "short" package name that is literally just the name of the package.
			// packageName is useful for location info in assertions, while shortPackageName is useful for
			// matching the names of packages in import statements when resolving wrapped assertions
			aScanner.shortPackageName = package_file.Name.Name
		}
	}
}

func (aScanner *AssertionScanner) setCurrentFunction(func_decl *ast.FuncDecl) {
	// Set the current function declaration
	aScanner.funcDecl = func_decl

	// Set the current function name
	aScanner.funcName = common.NAME_NOT_AVAILABLE
	if func_ident := func_decl.Name; func_ident != nil {
		aScanner.funcName = func_ident.Name
	}

	// Set the current function receiver
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

func (aScanner *AssertionScanner) setCurrentImports(import_spec *ast.ImportSpec) {
	assertPackageName := common.AssertPackageName()

	var path_name string
	path_name, _ = strconv.Unquote(import_spec.Path.Value)
	call_qualifier := path.Base(path_name)
	if import_spec.Name != nil && import_spec.Name.Name != "" {
		call_qualifier = import_spec.Name.Name
	}

	// If the imported package exactly matches the path of our assert package, then we store the name it was imported as
	if path_name == assertPackageName {
		aScanner.assertPackageImportNames = append(aScanner.assertPackageImportNames, call_qualifier)
	}

	// Store a map of "imported-as-name" to the original package path
	aScanner.importNameToPackageName[call_qualifier] = path.Base(path_name)
}

// First pass through the AST. This looks for any methods that call our
// wrapped assertion functions and saves information about them for the second pass.
// Specifically, how this works is that in this pass, we record the list of (wrapperMethodName, wrapperMethodPackageName, antithesisAssertionName)
// tuples that we see. (We really record this as a nested dictionary for efficiency).
// Then, during the second pass, we catalog any calls we find to each wrapperMethodPackageName.wrapperMethodName() as the associated antithesisAssertionName
func (aScanner *AssertionScanner) node_preprocessor(x ast.Node) bool {
	var call_expr *ast.CallExpr
	var func_decl *ast.FuncDecl
	var import_spec *ast.ImportSpec
	var ok bool

	// Get the package name for this file, if we haven't yet
	aScanner.setCurrentPackageName(x)

	// Collect the imports. We'll need this during this pass so that we can track aliases of
	// the assert package, so that we can find any methods that call our wrapped assertions
	if import_spec, ok = x.(*ast.ImportSpec); ok {
		aScanner.setCurrentImports(import_spec)
	}

	// Track current funcName and receiver (type)
	if func_decl, ok = x.(*ast.FuncDecl); ok {
		aScanner.setCurrentFunction(func_decl)
	}

	// If we hit a function call expression, we need to see if that function call represents a call to one of our wrapped
	// methods. If so, we'll record the function it's being called from as a wrapper function.
	if call_expr, ok = x.(*ast.CallExpr); ok {
		// We expect any call to a wrapped method to be a selector expression (e.g. assert.AlwaysInner(...))
		// Function call expressions can also be identity expressions (ast.Ident - e.g. AlwaysInner(...))
		// This could happen if people use dot-import, but we're not supporting this at this time, and is discouraged in general by the go docs.
		var sel_expr *ast.SelectorExpr
		if sel_expr, ok = call_expr.Fun.(*ast.SelectorExpr); ok {
			calledFunctionName := sel_expr.Sel.Name
			if assertFunctionName, ok := aScanner.innerToNormalAssertNames[calledFunctionName]; ok {
				aScanner.logWriter.PrintfIfVerbose(1, "Found assertion wrapper: %v wraps %v\n", calledFunctionName, assertFunctionName)

				isValid, parameterIndex := aScanner.getMessageParameter(aScanner.funcDecl)
				if !isValid {
					aScanner.logWriter.Fatalf("Wrapper %v is invalid - see earlier message for details. Exiting", aScanner.funcName)
				}

				if _, ok := aScanner.packageNameToWrapperNameToInfo[aScanner.shortPackageName][aScanner.funcName]; ok {
					aScanner.logWriter.Fatalf("Wrapper %v is invalid - wrappers can only contain one inner assertion", aScanner.funcName)
				}

				// Validate that the wrapper doesn't have a receiver. We could conceivably support this, but for the time being it makes the
				// implementation simpler since it makes the cataloging logic simpler in the second AST pass.
				if aScanner.funcDecl.Recv != nil && len(aScanner.funcDecl.Recv.List) > 0 {
					aScanner.logWriter.Fatalf("Wrapper %v is invalid - wrappers cannot take receivers", aScanner.funcName)
				}

				if _, ok := aScanner.packageNameToWrapperNameToInfo[aScanner.shortPackageName]; !ok {
					aScanner.packageNameToWrapperNameToInfo[aScanner.shortPackageName] = make(map[string]WrapperInfo)
				}

				aScanner.packageNameToWrapperNameToInfo[aScanner.shortPackageName][aScanner.funcName] = WrapperInfo{
					Assertion:  assertFunctionName,
					MessageArg: parameterIndex,
				}
			}
		}
	}

	return true
}

func (aScanner *AssertionScanner) node_inspector(x ast.Node) bool {
	var call_expr *ast.CallExpr
	var func_decl *ast.FuncDecl
	var import_spec *ast.ImportSpec
	var fun_expr ast.Expr
	var call_args []ast.Expr
	var ok bool

	// Get the package name for this file, if we haven't yet
	aScanner.setCurrentPackageName(x)

	// Collect the imports. We'll need this during this pass so that we can track aliases of
	// the assert package, so that we can find any methods that call our wrapped assertions
	if import_spec, ok = x.(*ast.ImportSpec); ok {
		aScanner.setCurrentImports(import_spec)
	}

	// Track current funcName and receiver (type)
	if func_decl, ok = x.(*ast.FuncDecl); ok {
		aScanner.setCurrentFunction(func_decl)
	}

	// When we find a function call expression, we want to see if that call represents a call to one of our assertion
	// methods OR one of the wrapper methods we found in the first AST pass. If so, catalog it.
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

			// If the function name matches the name of a function that we found to be a wrapper function, then we should catalog
			// every instance of that wrapper function.
			if wrapperNameToInfo, ok := aScanner.packageNameToWrapperNameToInfo[aScanner.shortPackageName]; ok {
				if wrapperInfo, ok := wrapperNameToInfo[call_name]; ok {
					aScanner.logWriter.PrintfIfVerbose(2, "Found an inner assertion: %v wraps %v\n", call_name, wrapperInfo.Assertion)

					full_position := aScanner.fset.Position(call_ident.Pos())
					relative_file_path := aScanner.module_relative_name(full_position.Filename)

					if wrapped_hint, ok := aScanner.assertionHintMap[wrapperInfo.Assertion]; ok {
						aScanner.addAssertion(wrapperInfo.Assertion, wrapped_hint, call_args, relative_file_path, full_position, wrapperInfo.MessageArg)
					} else if wrapped_guidance_hint, ok := aScanner.guidanceHintMap[wrapperInfo.Assertion]; ok {
						aScanner.addGuidance(wrapperInfo.Assertion, wrapped_guidance_hint, call_args, relative_file_path, full_position, wrapperInfo.MessageArg)
					} else {
						panic("Should never get here - couldn't find assertion or guidance hint")
					}
				}
			}

			if aScanner.logWriter.VerboseLevel(2) {
				aScanner.logWriter.Printf("Found call to %s()\n", call_name)
			}
		}

		var sel_expr *ast.SelectorExpr
		if sel_expr, ok = fun_expr.(*ast.SelectorExpr); ok {
			full_position := aScanner.fset.Position(sel_expr.Pos())
			relative_file_path := aScanner.module_relative_name(full_position.Filename)
			isAssertionPackage := doesSelectorMatchAssertionPackage(aScanner.assertPackageImportNames, sel_expr.X)
			calledFunctionName := sel_expr.Sel.Name

			// Normal (non-wrapped) assertions
			if isAssertionPackage {
				if func_hints := aScanner.assertionHintMap.HintsForName(calledFunctionName); func_hints != nil {
					aScanner.addAssertion(calledFunctionName, func_hints, call_args, relative_file_path, full_position, func_hints.MessageArg)
				}

				if guidance_func_hints := aScanner.guidanceHintMap.GuidanceHintsForName(calledFunctionName); guidance_func_hints != nil {
					aScanner.addGuidance(calledFunctionName, guidance_func_hints, call_args, relative_file_path, full_position, guidance_func_hints.MessageArg)
				}
			}

			// Inner assertions
			selectorName := getSelectorExpressionName(sel_expr.X)
			if packageName, ok := aScanner.importNameToPackageName[selectorName]; ok {
				if wrapperNameToInfo, ok := aScanner.packageNameToWrapperNameToInfo[packageName]; ok {
					if wrapperInfo, ok := wrapperNameToInfo[calledFunctionName]; ok {
						aScanner.logWriter.PrintfIfVerbose(2, "Found inner assertion: %v wraps %v\n", calledFunctionName, wrapperInfo.Assertion)

						if wrapped_hint, ok := aScanner.assertionHintMap[wrapperInfo.Assertion]; ok {
							aScanner.addAssertion(wrapperInfo.Assertion, wrapped_hint, call_args, relative_file_path, full_position, wrapperInfo.MessageArg)
						} else if wrapped_guidance_hint, ok := aScanner.guidanceHintMap[wrapperInfo.Assertion]; ok {
							aScanner.addGuidance(wrapperInfo.Assertion, wrapped_guidance_hint, call_args, relative_file_path, full_position, wrapperInfo.MessageArg)
						} else {
							panic("Should never get here - couldn't find assertion or guidance hint")
						}
					}
				}
			}
		}
	}
	return true
}

func (aScanner *AssertionScanner) addGuidance(assertion_function_name string, guidance_hint *GuidanceFuncInfo, call_args []ast.Expr, relative_file_path string, full_position token.Position, messageArg int) {
	test_name := arg_at_index(call_args, messageArg)
	if test_name == common.NAME_NOT_AVAILABLE {
		generated_msg := fmt.Sprintf("%s[%d]", relative_file_path, full_position.Line)
		test_name = fmt.Sprintf("Message from %s", strconv.Quote(generated_msg))
	}

	// The registration for the Guidance function itself
	guidance_expect := AntGuidance{
		Assertion:        assertion_function_name,
		Message:          test_name,
		Classname:        aScanner.packageName,
		Funcname:         aScanner.funcName,
		Receiver:         aScanner.receiver,
		Filename:         relative_file_path,
		Line:             full_position.Line,
		GuidanceFuncInfo: guidance_hint,
	}
	aScanner.guidance = append(aScanner.guidance, &guidance_expect)

	// The Related Assertion derived from target_func("AlwaysGreaterThan") => derived_target_func("Always")
	expect := AntExpect{
		Assertion: target_func_from_guidance(assertion_function_name),
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
		AssertionFuncInfo: &guidance_hint.AssertionFuncInfo,
	}
	aScanner.expects = append(aScanner.expects, &expect)
}

func (aScanner *AssertionScanner) addAssertion(assertion_function_name string, assertion_hint *AssertionFuncInfo, call_args []ast.Expr, relative_file_path string, full_position token.Position, messageArg int) {
	test_name := arg_at_index(call_args, messageArg)
	if test_name == common.NAME_NOT_AVAILABLE {
		generated_msg := fmt.Sprintf("%s[%d]", relative_file_path, full_position.Line)
		test_name = fmt.Sprintf("Message from %s", strconv.Quote(generated_msg))
	}

	expect := AntExpect{
		Assertion:         assertion_function_name,
		Message:           test_name,
		Classname:         aScanner.packageName,
		Funcname:          aScanner.funcName,
		Receiver:          aScanner.receiver,
		Filename:          relative_file_path,
		Line:              full_position.Line,
		AssertionFuncInfo: assertion_hint,
	}

	aScanner.expects = append(aScanner.expects, &expect)
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

func doesSelectorMatchAssertionPackage(imports []string, expr ast.Expr) bool {
	expr_name := getSelectorExpressionName(expr)

	for _, import_name := range imports {
		if import_name == expr_name {
			return true
		}
	}
	return false
}

func getSelectorExpressionName(expr ast.Expr) string {
	if expr_id, ok := expr.(*ast.Ident); ok {
		return expr_id.Name
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

func (aScanner *AssertionScanner) LogAssertionWrappers() {
	if aScanner.logWriter.VerboseLevel(1) {
		for k, v := range aScanner.packageNameToWrapperNameToInfo {
			for k2, v2 := range v {
				aScanner.logWriter.Printf("Wrapper map entry (package, wrapperMethodName, assertion, messageArg) (%v, %v, %v, %v)", k, k2, v2.Assertion, v2.MessageArg)
			}
		}
	}
}

func SetupInnerAssertionMap() map[string]string {
	return map[string]string{
		// Basic assertions
		"AlwaysInner":              "Always",
		"AlwaysOrUnreachableInner": "AlwaysOrUnreachable",
		"SometimesInner":           "Sometimes",
		"UnreachableInner":         "Unreachable",
		"ReachableInner":           "Reachable",

		// Rich assertions
		"AlwaysGreaterThanInner":             "AlwaysGreaterThan",
		"AlwaysGreaterThanOrEqualToInner":    "AlwaysGreaterThanOrEqualTo",
		"SometimesGreaterThanInner":          "SometimesGreaterThan",
		"SometimesGreaterThanOrEqualToInner": "SometimesGreaterThanOrEqualTo",
		"AlwaysLessThanInner":                "AlwaysLessThan",
		"AlwaysLessThanOrEqualToInner":       "AlwaysLessThanOrEqualTo",
		"SometimesLessThanInner":             "SometimesLessThan",
		"SometimesLessThanOrEqualToInner":    "SometimesLessThanOrEqualTo",
		"AlwaysSomeInner":                    "AlwaysSome",
		"SometimesAllInner":                  "SometimesAll",
	}
}

func (aScanner *AssertionScanner) getMessageParameter(fd *ast.FuncDecl) (bool, int) {
	if fd == nil {
		aScanner.logWriter.Printf("Wrapper function must have function declaration\n")
		return false, -1
	}

	fieldList := fd.Type.Params
	if fieldList == nil {
		aScanner.logWriter.Printf("Wrapper function must have a message parameter\n")
		return false, -1
	}

	// The outer list is a list of parameter groups. Usually a group has a single parameter, but a group has multiple parameters
	// when they share a type expression [e.g. func foo(x, y int)]
	currentIndex := 0
	for _, f := range fieldList.List {
		// If a field has no names, it's an unnamed parameter
		if len(f.Names) == 0 {
			currentIndex++
		} else {
			// Otherwise, iterate through the parameters in the group until we find the message parameter or we exhaust this group
			for _, name := range f.Names {
				if name.Name == "message" {
					// Check if the message parameter is a string, return false if so
					if messageTypeidentifier, ok := f.Type.(*ast.Ident); ok {
						if messageTypeidentifier.Name == "string" {
							return true, currentIndex
						}

						aScanner.logWriter.Printf("Message parameter must be a string")
						return false, -1
					} else {
						aScanner.logWriter.Printf("Message parameter must be a string")
						return false, -1
					}
				}
				currentIndex++
			}
		}
	}

	aScanner.logWriter.Printf("Wrapper function must have a message parameter\n")
	return false, -1
}
