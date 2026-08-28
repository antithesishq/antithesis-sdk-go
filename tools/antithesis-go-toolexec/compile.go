package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"

	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/scanners/coverage/instrumentor"
	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/scanners/coverage/symboltable"
)

// registrationFile is the name given to the file we synthesise into the package
// being compiled. It never reaches the user's tree.
const registrationFile = "antithesis_module.go"

// rewriteCompile applies the enabled features to a compile invocation. Coverage
// instrumentation and assertion cataloguing are independent: either can be turned
// off without affecting the other.
//
// At most three edits are made, and every other argument -- -o, -p, -lang,
// -buildid, -trimpath, -complete, -embedcfg, -pack -- is passed through:
//
//  1. each source file path is replaced with an instrumented copy;
//  2. generated files are appended (a coverage registration, a catalog); and
//  3. -importcfg is replaced with a copy that can resolve the SDK.
//
// Nothing is written into the user's source tree, and nothing outlives the build
// except the package's symbol table.
//
// The returned cleanup function removes the generated files. It must not run
// until the compiler has finished reading them.
func rewriteCompile(cfg *Config, pkg *Package, args []string) ([]string, func(), error) {
	work, err := os.MkdirTemp("", "antithesis-instr-")
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { os.RemoveAll(work) }
	// Every failure below must release the work directory: the caller only gets
	// the cleanup function on success.
	fail := func(err error) ([]string, func(), error) {
		cleanup()
		return nil, nil, err
	}

	paths := make([]string, 0, len(pkg.GoFiles))
	for _, file := range pkg.GoFiles {
		paths = append(paths, file.Path)
	}

	// Cataloguing reads the original sources and the compiler's own importcfg, so
	// it has to run before coverage rewrites either.
	if cfg.Catalog {
		catalog, err := catalogPackage(cfg, pkg, args, paths, work)
		if err != nil {
			return fail(err)
		}
		if catalog != "" {
			args = append(args, catalog)
		}
	}

	if cfg.Coverage {
		if args, err = instrumentCoverage(cfg, pkg, args, paths, work); err != nil {
			return fail(err)
		}
	}
	return args, cleanup, nil
}

// instrumentCoverage adds coverage edges to the package being compiled: it
// rewrites the sources, writes the package's symbol table, and injects the
// registration that binds the two at run time.
func instrumentCoverage(cfg *Config, pkg *Package, args []string, paths []string, work string) ([]string, error) {
	// The symbol table's name is derived from this package's own sources, so it is
	// stable across builds and unaffected by changes elsewhere in the tree.
	digest := common.HashFileContent(paths)
	symbolBase := cfg.SymbolPrefix + common.SYMBOLS_FILE_HASH_PREFIX + "-" + digest
	symbolName := symbolBase + common.SYMBOLS_FILE_SUFFIX

	shardDir, err := shardDirectory()
	if err != nil {
		return nil, err
	}
	symbols, err := symboltable.CreateSymbolTableFile(filepath.Join(shardDir, symbolName), symbolBase)
	if err != nil {
		return nil, fmt.Errorf("creating symbol table: %w", err)
	}

	// basePath controls only the path in the //line directive the instrumentor
	// emits; symbol table rows always record the absolute path. Pass the module
	// root, as the whole-tree instrumentor does, so the two front ends agree on
	// what //line says.
	edges := instrumentor.CreateSourceEditingInstrumentor(moduleRoot(filepath.Dir(paths[0])), "", symbols)

	// The call sites and the registration below must agree on the identifier, and
	// it has to be one this package does not already use; see callback.go.
	callbackName, err := resolveCallbackName(paths)
	if err != nil {
		return nil, err
	}
	if callbackName != instrumentor.InstrumentationPackageAlias {
		common.Logger.Printf(common.Normal, "%s already declares %s; using %s instead",
			pkg.ImportPath, instrumentor.InstrumentationPackageAlias, callbackName)
	}
	edges.SetCallbackName(callbackName)

	instrumented := 0
	for _, file := range pkg.GoFiles {
		source, err := edges.Instrument(file.Path)
		if err != nil {
			// A file we cannot rewrite is still a file the package needs, so leave
			// the original in place and carry on with the rest.
			common.Logger.Printf(common.Normal, "Error: %s not instrumented (%s); using original", file.Path, err)
			continue
		}
		if source == "" {
			// The instrumentor declined this file: no edges, or it exports functions
			// or declares linknames. Its original path stays in the argument list.
			continue
		}
		rewritten := filepath.Join(work, fmt.Sprintf("%d_%s", file.Index, filepath.Base(file.Path)))
		if err := common.WriteTextFile(source, rewritten); err != nil {
			return nil, fmt.Errorf("writing %s: %w", rewritten, err)
		}
		args[file.Index] = rewritten // edit 1
		instrumented++
	}

	edgeCount := edges.Edges()
	if err := symbols.Close(); err != nil {
		return nil, fmt.Errorf("closing symbol table: %w", err)
	}
	if edgeCount == 0 {
		// Nothing was instrumented after all; drop the empty symbol table rather
		// than leaving a header-only file to be collected into the image.
		os.Remove(filepath.Join(shardDir, symbolName))
		return args, nil
	}

	packageName, err := packageClause(pkg.GoFiles[0].Path)
	if err != nil {
		return nil, err
	}
	registration := filepath.Join(work, registrationFile)
	if err := common.WriteTextFile(registrationSource(packageName, callbackName, symbolName, edgeCount), registration); err != nil {
		return nil, fmt.Errorf("writing %s: %w", registration, err)
	}
	args = append(args, registration) // edit 2

	if err := recordShard(shardDir, pkg.ImportPath, symbolName); err != nil {
		return nil, err
	}

	// edit 3
	if args, err = ensureSDKImportable(cfg, args, pkg.ImportCfg); err != nil {
		return nil, err
	}

	common.Logger.Printf(common.Info, "Instrumented %s: %d %s in %d %s -> %s",
		pkg.ImportPath, edgeCount, common.Pluralize(edgeCount, "edge"),
		instrumented, common.Pluralize(instrumented, "file"), symbolName)
	return args, nil
}

// registrationSource is the file we add to the package being compiled.
//
// The variable is named for the identifier the instrumentor's call sites use, so
// `<callbackName>.Notify(n)` resolves to a method on this handle exactly as it
// would have resolved to a function in a notifier package.
//
// Declaring it as a package-level variable orders initialization correctly.
// Go's dependency analysis ensures that the SDK and this variable are initialized
// before any variable in the package whose initializer calls instrumented code.
func registrationSource(packageName, callbackName, symbolName string, edgeCount int) string {
	return fmt.Sprintf(`// Code generated by antithesis-go-toolexec. DO NOT EDIT.

package %s

import %s %q

var %s = %s.RegisterModule(%q, %d)
`,
		packageName,
		sdkImportAlias, sdkInstrumentationPackage,
		callbackName, sdkImportAlias,
		symbolName, edgeCount)
}

// sdkImportAlias keeps the SDK import from colliding with anything the package
// already imports under its own name.
const sdkImportAlias = "__antithesis_sdk__"

// packageClause reads just the package name, which the compile invocation does
// not tell us: -p carries the import path, and the two differ often enough
// (external test packages, packages named differently from their directory) that
// it cannot be guessed.
func packageClause(path string) (string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly)
	if err != nil {
		return "", fmt.Errorf("reading package clause from %s: %w", path, err)
	}
	return file.Name.Name, nil
}
