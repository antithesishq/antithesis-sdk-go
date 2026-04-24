package assertions

import (
	"fmt"
	"go/ast"
	"go/types"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
	"golang.org/x/tools/go/packages"
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

// AssertionScanner scans Go packages using go/packages to find assertion
// and guidance calls, producing per-binary catalog data.
type AssertionScanner struct {
	assertionHintMap AssertionHints
	guidanceHintMap  GuidanceHints
	baseInputDir     string
	baseTargetDir    string
	filesCataloged   int

	// Results: per-binary catalogs
	binaries []*binaryCatalog
}

// binaryCatalog represents a discovered main package and its reachable assertions.
type binaryCatalog struct {
	dir      string // absolute path of the main package directory
	relDir   string // directory relative to the module root
	expects  []*AntExpect
	guidance []*AntGuidance
}

func (bc *binaryCatalog) hasAssertions() bool {
	return len(bc.expects) > 0 || len(bc.guidance) > 0
}

// packageResult caches the scan result for a single package.
type packageResult struct {
	expects  []*AntExpect
	guidance []*AntGuidance
}

// filter Guidance to just numeric
func numericGuidance(guidance []*AntGuidance) []*AntGuidance {
	numeric_guidance := []*AntGuidance{}
	for _, aG := range guidance {
		gp := aG.GuidanceFn
		if gp == GuidanceFnMaximize || gp == GuidanceFnMinimize {
			numeric_guidance = append(numeric_guidance, aG)
		}
	}
	return numeric_guidance
}

// filter Guidance to just boolean
func booleanGuidance(guidance []*AntGuidance) []*AntGuidance {
	boolean_guidance := []*AntGuidance{}
	for _, aG := range guidance {
		gp := aG.GuidanceFn
		if gp == GuidanceFnWantAll || gp == GuidanceFnWantNone {
			boolean_guidance = append(boolean_guidance, aG)
		}
	}
	return boolean_guidance
}

func NewAssertionScanner(sourceDir string, targetDir string) *AssertionScanner {
	aScanner := AssertionScanner{
		baseInputDir:     sourceDir,
		baseTargetDir:    targetDir,
		assertionHintMap: SetupHintMap(),
		guidanceHintMap:  SetupGuidanceHintMap(),
		filesCataloged:   0,
	}
	return &aScanner
}

// HasAssertionsDefined returns true if any binary has assertions.
func (aScanner *AssertionScanner) HasAssertionsDefined() bool {
	for _, bc := range aScanner.binaries {
		if bc.hasAssertions() {
			return true
		}
	}
	return false
}

// WriteAssertionCatalogs writes one catalog per binary into the appropriate
// directory under baseTargetDir. For assert-only mode, baseTargetDir ==
// baseInputDir.
func (aScanner *AssertionScanner) WriteAssertionCatalogs(versionText string) {
	now := time.Now()
	createDate := now.Format("Mon Jan 2 15:04:05 MST 2006")

	for _, bc := range aScanner.binaries {
		if !bc.hasAssertions() {
			continue
		}

		expects := bc.expects
		numericGuidance := numericGuidance(bc.guidance)
		booleanGuidance := booleanGuidance(bc.guidance)

		genInfo := GenInfo{
			ExpectedVals:        expects,
			NumericGuidanceVals: numericGuidance,
			BooleanGuidanceVals: booleanGuidance,
			AssertPackageName:   common.AssertPackageName(),
			VersionText:         versionText,
			CreateDate:          createDate,
			HasAssertions:       len(expects) > 0,
			HasNumericGuidance:  len(numericGuidance) > 0,
			HasBooleanGuidance:  len(booleanGuidance) > 0,
			ConstMap:            getConstMap(expects),
		}

		outputDir := filepath.Join(aScanner.baseTargetDir, bc.relDir)
		GenerateAssertionsCatalog(outputDir, &genInfo)
	}
}

func (aScanner *AssertionScanner) SummarizeWork() {
	numCataloged := aScanner.filesCataloged
	common.Logger.Printf(common.Normal, "%d '.go' %s cataloged", numCataloged, common.Pluralize(numCataloged, "file"))
	numBinaries := len(aScanner.binaries)
	catalogsWritten := 0
	for _, bc := range aScanner.binaries {
		if bc.hasAssertions() {
			catalogsWritten++
		}
	}
	common.Logger.Printf(common.Normal, "%d %s discovered, %d %s written",
		numBinaries, common.Pluralize(numBinaries, "binary"),
		catalogsWritten, common.Pluralize(catalogsWritten, "catalog"))
}

// ScanAll loads all packages under baseInputDir using go/packages, identifies
// main packages, computes per-binary reachability, and scans for assertions.
func (aScanner *AssertionScanner) ScanAll() error {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedSyntax |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedModule,
		Dir: aScanner.baseInputDir,
	}

	common.Logger.Printf(common.Info, "Loading packages from %s", aScanner.baseInputDir)

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return fmt.Errorf("go/packages load failed: %w", err)
	}

	// Check for package errors
	var loadErrors []string
	packages.Visit(pkgs, nil, func(pkg *packages.Package) {
		for _, e := range pkg.Errors {
			loadErrors = append(loadErrors, fmt.Sprintf("%s: %s", pkg.PkgPath, e.Msg))
		}
	})
	if len(loadErrors) > 0 {
		return fmt.Errorf("package load errors:\n  %s", strings.Join(loadErrors, "\n  "))
	}

	// Identify main packages
	var mainPkgs []*packages.Package
	for _, pkg := range pkgs {
		if pkg.Name == "main" {
			mainPkgs = append(mainPkgs, pkg)
		}
	}

	if len(mainPkgs) == 0 {
		common.Logger.Printf(common.Normal, "Warning: no main packages found in %s", aScanner.baseInputDir)
		return nil
	}

	common.Logger.Printf(common.Info, "Found %d main %s", len(mainPkgs), common.Pluralize(len(mainPkgs), "package"))
	for _, pkg := range mainPkgs {
		common.Logger.Printf(common.Info, "  main: %s", pkg.PkgPath)
	}

	// For each main package, compute reachable packages and scan.
	// Cache per-package results so shared dependencies are only scanned once.
	assertPkgPath := common.AssertPackageName()
	pkgCache := make(map[string]*packageResult)

	for _, mainPkg := range mainPkgs {
		common.Logger.Printf(common.Normal, "Cataloging %s", mainPkg.PkgPath)
		reachable := collectReachable(mainPkg)

		bc := &binaryCatalog{
			dir:    mainPkg.Dir,
			relDir: aScanner.relativeDir(mainPkg.Dir),
		}
		for _, pkg := range reachable {
			pr, ok := pkgCache[pkg.ID]
			if !ok {
				pr = &packageResult{}
				common.Logger.Printf(common.Info, "Scanning %s", pkg.PkgPath)
				for _, file := range pkg.Syntax {
					aScanner.filesCataloged++
					funcName := ""
					receiver := ""
					ast.Inspect(file, func(x ast.Node) bool {
						funcName, receiver = aScanner.node_inspector(x, pkg, assertPkgPath, pr, funcName, receiver)
						return true
					})
				}
				pkgCache[pkg.ID] = pr
			}
			bc.expects = append(bc.expects, pr.expects...)
			bc.guidance = append(bc.guidance, pr.guidance...)
		}
		aScanner.binaries = append(aScanner.binaries, bc)

		common.Logger.Printf(common.Info, "Binary %s: %d assertions, %d guidance entries",
			bc.relDir, len(bc.expects), len(bc.guidance))
	}

	return nil
}

func (aScanner *AssertionScanner) node_inspector(x ast.Node, pkg *packages.Package, assertPkgPath string, data *packageResult, funcName string, receiver string) (string, string) {
	var func_decl *ast.FuncDecl
	var call_expr *ast.CallExpr
	var ok bool

	// Track current funcName and receiver (type)
	if func_decl, ok = x.(*ast.FuncDecl); ok {
		funcName = common.NAME_NOT_AVAILABLE
		if func_ident := func_decl.Name; func_ident != nil {
			funcName = func_ident.Name
		}
		receiver = ""
		if recv := func_decl.Recv; recv != nil {
			if num_fields := recv.NumFields(); num_fields > 0 {
				if field_list := recv.List; field_list != nil {
					if recv_type := field_list[0].Type; recv_type != nil {
						receiver = types.ExprString(recv_type)
					}
				}
			}
		}
	}

	if call_expr, ok = x.(*ast.CallExpr); ok {
		var sel_expr *ast.SelectorExpr
		if sel_expr, ok = call_expr.Fun.(*ast.SelectorExpr); ok {
			// Use type info to resolve the called function
			// in the SDK assert package
			obj := pkg.TypesInfo.Uses[sel_expr.Sel]
			if obj == nil {
				return funcName, receiver
			}

			fn, ok := obj.(*types.Func)
			if !ok {
				return funcName, receiver
			}

			if fn.Pkg() == nil || fn.Pkg().Path() != assertPkgPath {
				return funcName, receiver
			}

			target_func := fn.Name()
			full_position := pkg.Fset.Position(sel_expr.Pos())
			relative_file_path := aScanner.relativeDir(full_position.Filename)
			packageName := pkg.PkgPath
			call_args := call_expr.Args

			if func_hints := aScanner.assertionHintMap.HintsForName(target_func); func_hints != nil {
				test_name := arg_at_index(call_args, func_hints.MessageArg)
				if test_name == common.NAME_NOT_AVAILABLE {
					generated_msg := fmt.Sprintf("%s[%d]", relative_file_path, full_position.Line)
					test_name = fmt.Sprintf("Message from %s", strconv.Quote(generated_msg))
				}
				expect := AntExpect{
					Assertion:         target_func,
					Message:           test_name,
					Classname:         packageName,
					Funcname:          funcName,
					Receiver:          receiver,
					Filename:          relative_file_path,
					Line:              full_position.Line,
					AssertionFuncInfo: func_hints,
				}
				data.expects = append(data.expects, &expect)
			}

			if guidance_func_hints := aScanner.guidanceHintMap.GuidanceHintsForName(target_func); guidance_func_hints != nil {
				test_name := arg_at_index(call_args, guidance_func_hints.MessageArg)
				if test_name == common.NAME_NOT_AVAILABLE {
					generated_msg := fmt.Sprintf("%s[%d]", relative_file_path, full_position.Line)
					test_name = fmt.Sprintf("Message from %s", strconv.Quote(generated_msg))
				}
				// The registration for the Guidance function itself
				guidance_expect := AntGuidance{
					Assertion:        target_func,
					Message:          test_name,
					Classname:        packageName,
					Funcname:         funcName,
					Receiver:         receiver,
					Filename:         relative_file_path,
					Line:             full_position.Line,
					GuidanceFuncInfo: guidance_func_hints,
				}
				data.guidance = append(data.guidance, &guidance_expect)

				// The Related Assertion derived from target_func("AlwaysGreaterThan") => derived_target_func("Always")
				expect := AntExpect{
					Assertion: target_func_from_guidance(target_func),
					Message:   test_name,
					Classname: packageName,
					Funcname:  funcName,
					Receiver:  receiver,
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
				data.expects = append(data.expects, &expect)
			} // assertionHint
		}
	}
	return funcName, receiver
}

func (aScanner *AssertionScanner) relativeDir(dir string) string {
	rel, err := filepath.Rel(aScanner.baseInputDir, dir)
	if err != nil {
		return dir
	}
	return rel
}

// collectReachable returns all packages transitively reachable from pkg
// that belong to the same module. Packages from the standard library or
// other modules are skipped.
func collectReachable(pkg *packages.Package) []*packages.Package {
	if pkg.Module == nil {
		return nil
	}
	modulePath := pkg.Module.Path

	visited := make(map[string]bool)
	var result []*packages.Package

	var walk func(p *packages.Package)
	walk = func(p *packages.Package) {
		if visited[p.ID] {
			return
		}
		visited[p.ID] = true
		if p.Module == nil || p.Module.Path != modulePath {
			return
		}
		result = append(result, p)
		for _, imp := range p.Imports {
			walk(imp)
		}
	}
	walk(pkg)
	return result
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
func getConstMap(expects []*AntExpect) map[string]bool {
	cond_tracker := make([]bool, Num_conditions)
	if len(expects) > 0 {
		cond_tracker[Not_hit] = true
	}
	for _, an_expect := range expects {
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
