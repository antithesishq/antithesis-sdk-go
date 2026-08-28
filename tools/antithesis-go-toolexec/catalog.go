package main

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/scanners/assertions"
)

const catalogFile = common.GENERATED_CATALOG_FILE

// catalogPackage writes an assertion catalog for the package being compiled and
// returns its path, or "" if the package has no assertions.
//
// sourcePaths are the package's original files, not the instrumented rewrites:
// cataloguing records source positions, and there is no reason to read them
// through a transformation.
func catalogPackage(cfg *Config, pkg *Package, args []string, sourcePaths []string, work string) (string, error) {
	fset := token.NewFileSet()
	files := make([]*ast.File, 0, len(sourcePaths))
	importsAssert := false
	for _, path := range sourcePaths {
		parsed, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return "", fmt.Errorf("parsing %s: %w", path, err)
		}
		for _, spec := range parsed.Imports {
			if strings.Trim(spec.Path.Value, `"`) == common.AssertPackageName() {
				importsAssert = true
			}
		}
		files = append(files, parsed)
	}

	// A package that does not import the assert package cannot contain an
	// assertion, so there is nothing to find and no reason to pay for a
	// type-check. This is the common case by a wide margin.
	if !importsAssert {
		return "", nil
	}

	info, err := typeCheck(pkg.ImportPath, fset, files, args)
	if err != nil {
		return "", err
	}

	// Record positions relative to the module root, as the whole-tree
	// instrumentor does, so catalogs do not carry the build machine's paths.
	scanner := assertions.NewAssertionScanner(moduleRoot(filepath.Dir(sourcePaths[0])), "")
	result := scanner.ScanPackage(assertions.PackageSyntax{
		PkgPath:   pkg.ImportPath,
		Fset:      fset,
		TypesInfo: info,
		Files:     files,
	})
	if !result.HasAssertions() {
		return "", nil
	}

	packageName, err := packageClause(sourcePaths[0])
	if err != nil {
		return "", err
	}
	if err := assertions.GenerateAssertionsCatalog(work, catalogInfo(packageName, result)); err != nil {
		return "", err
	}
	path := filepath.Join(work, catalogFile)
	common.Logger.Printf(common.Info, "Cataloged %s: %d %s, %d guidance %s",
		pkg.ImportPath,
		len(result.Expects), common.Pluralize(len(result.Expects), "assertion"),
		len(result.Guidance), common.Pluralize(len(result.Guidance), "record"))
	return path, nil
}

func catalogInfo(packageName string, result *assertions.PackageResult) *assertions.GenInfo {
	numeric := assertions.NumericGuidance(result.Guidance)
	boolean := assertions.BooleanGuidance(result.Guidance)
	return &assertions.GenInfo{
		PackageName:         packageName,
		ExpectedVals:        result.Expects,
		NumericGuidanceVals: numeric,
		BooleanGuidanceVals: boolean,
		AssertPackageName:   common.AssertPackageName(),
		VersionText:         "antithesis-go-toolexec",
		CreateDate:          time.Now().Format("Mon Jan 2 15:04:05 MST 2006"),
		HasAssertions:       len(result.Expects) > 0,
		HasNumericGuidance:  len(numeric) > 0,
		HasBooleanGuidance:  len(boolean) > 0,
		ConstMap:            assertions.ConstMap(result.Expects),
	}
}

// resolves the types for this package using the archives named in the importcfg
// the compiler was handed
func typeCheck(importPath string, fset *token.FileSet, files []*ast.File, args []string) (*types.Info, error) {
	importCfg := flagValue(args, "-importcfg")
	if importCfg == "" {
		return nil, fmt.Errorf("no -importcfg in the compile invocation for %s", importPath)
	}
	cfg, err := readImportCfg(importCfg)
	if err != nil {
		return nil, err
	}

	config := types.Config{
		Importer: importer.ForCompiler(fset, "gc", func(path string) (io.ReadCloser, error) {
			archive, ok := cfg.archives[path]
			if !ok {
				return nil, fmt.Errorf("no archive for %s in %s", path, importCfg)
			}
			return os.Open(archive)
		}),
		GoVersion: languageVersion(args),
		// Sizes are only consulted for constant overflow
		Sizes: types.SizesFor("gc", buildArch()),
	}
	info := &types.Info{
		Uses: map[*ast.Ident]types.Object{},
		Defs: map[*ast.Ident]types.Object{},
	}
	if _, err := config.Check(importPath, fset, files, info); err != nil {
		return nil, fmt.Errorf("type-checking %s for cataloguing: %w", importPath, err)
	}
	return info, nil
}

// languageVersion recovers the -lang value the compiler was given, in the form
// go/types expects.
func languageVersion(args []string) string {
	lang := flagValue(args, "-lang")
	if lang == "" {
		return ""
	}
	if !strings.HasPrefix(lang, "go") {
		lang = "go" + lang
	}
	return lang
}

// flagValue reads a flag written either as "-name value" or "-name=value".
func flagValue(args []string, name string) string {
	for i, arg := range args {
		if arg == name && i+1 < len(args) {
			return args[i+1]
		}
		if value, ok := strings.CutPrefix(arg, name+"="); ok {
			return value
		}
	}
	return ""
}

// buildArch reports the GOARCH this build targets
func buildArch() string {
	if arch := strings.TrimSpace(os.Getenv("GOARCH")); arch != "" {
		return arch
	}
	return runtime.GOARCH
}
