package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/scanners/coverage/instrumentor"
)

// resolveCallbackName returns an identifier free of collisions in the package
// whose files are paths.
func resolveCallbackName(paths []string) (string, error) {
	taken, err := packageScopeNames(paths)
	if err != nil {
		return "", err
	}
	for attempt := 0; attempt <= math.MaxInt; attempt++ {
		if candidate := callbackCandidate(attempt); !taken[candidate] {
			return candidate, nil
		}
	}
	// a user package would need to declare math.MaxInt + 1 variables that conflict with candidates in order to reach this error
	// it's certainly possible for an adversarial package to do that (hence the handling), but it should never occur in practice
	return "", fmt.Errorf(
		"could not find a free name for the coverage callback: %s and %d suffixed variants are all declared in this package",
		instrumentor.InstrumentationPackageAlias, math.MaxInt)
}

// callbackCandidate names the nth choice. The zeroth is the default the
// whole-tree instrumentor also uses, so an ordinary package is instrumented
// identically by both front ends.
func callbackCandidate(n int) string {
	base := instrumentor.InstrumentationPackageAlias
	if n == 0 {
		return base
	}
	// Keep the trailing underscores, so every variant still reads as generated.
	return strings.TrimSuffix(base, "__") + "_" + strconv.Itoa(n) + "__"
}

// packageScopeNames collects every identifier that would collide with a
// package-level variable: top-level declarations from any file, and the names
// files bind with their imports.
func packageScopeNames(paths []string) (map[string]bool, error) {
	prefix := []byte(strings.TrimSuffix(instrumentor.InstrumentationPackageAlias, "__"))
	taken := map[string]bool{}
	for _, path := range paths {
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if !bytes.Contains(body, prefix) {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, body, parser.SkipObjectResolution)
		if err != nil {
			// The compiler is about to report this properly; nothing to add.
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		collectDeclaredNames(file, taken)
	}
	return taken, nil
}

func collectDeclaredNames(file *ast.File, taken map[string]bool) {
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			// Methods are named in the type's scope, not the package's.
			if d.Recv == nil && d.Name != nil {
				taken[d.Name.Name] = true
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.ValueSpec:
					for _, name := range s.Names {
						taken[name.Name] = true
					}
				case *ast.TypeSpec:
					if s.Name != nil {
						taken[s.Name.Name] = true
					}
				case *ast.ImportSpec:
					if name := importBinding(s); name != "" {
						taken[name] = true
					}
				}
			}
		}
	}
}

// importBinding is the identifier an import introduces into its file's scope.
// Without an explicit alias the binding is the imported package's name, which
// cannot be known from the path alone -- but a package whose name matches a
// candidate would have to be imported from a path containing it, so the last
// path element is a sufficient approximation here.
func importBinding(spec *ast.ImportSpec) string {
	if spec.Name != nil {
		if spec.Name.Name == "_" || spec.Name.Name == "." {
			return ""
		}
		return spec.Name.Name
	}
	path, err := strconv.Unquote(spec.Path.Value)
	if err != nil {
		return ""
	}
	if index := strings.LastIndexByte(path, '/'); index >= 0 {
		return path[index+1:]
	}
	return path
}
