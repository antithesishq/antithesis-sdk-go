package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
)

// Package is what we learn about the package being compiled, from the compile
// invocation itself.
type Package struct {
	// ImportPath comes from TOOLEXEC_IMPORTPATH, which the go command sets to the
	// package being built.
	ImportPath string

	// GoFiles are the source files from the end of the argument list, carrying the
	// index each was found at so rewritten copies can be substituted in place.
	GoFiles []fileArg

	// ImportCfg is the index of the value following -importcfg.
	ImportCfg int
}

type fileArg struct {
	Index int
	Path  string
}

// selectedPackage decides whether this compile invocation is one we act on, and
// extracts what we need from its arguments if so. The same selection governs both
// coverage instrumentation and cataloguing.
//
// Three categories are always skipped:
//
//   - The standard library. The go command marks those compiles with -std. Some
//     of them (runtime, internal/abi) cannot tolerate a Notify call at all, and
//     instrumenting them would also mean instrumenting the code paths our own
//     notifications travel through.
//   - The SDK itself, for that same reentrancy reason.
//   - Anything outside the configured scope; see inScope.
func selectedPackage(cfg *Config, args []string) (*Package, bool) {
	importPath := os.Getenv("TOOLEXEC_IMPORTPATH")
	if importPath == "" {
		// No import path means the go command is doing something other than
		// building a package we could instrument.
		return nil, false
	}
	if slices.Contains(args, "-std") || isSDKPackage(importPath) {
		return nil, false
	}

	pkg := &Package{ImportPath: importPath, ImportCfg: -1, GoFiles: sourceFiles(args)}
	for i, arg := range args {
		if arg == "-importcfg" && i+1 < len(args) {
			pkg.ImportCfg = i + 1
		}
	}
	if len(pkg.GoFiles) == 0 || pkg.ImportCfg < 0 {
		return nil, false
	}
	if !inScope(cfg, importPath, pkg.GoFiles[0].Path) {
		return nil, false
	}

	pkg.GoFiles = slices.DeleteFunc(pkg.GoFiles, func(f fileArg) bool {
		if skip := skipFile(cfg, f.Path); skip {
			common.Logger.Printf(common.Info, "Skipping %s", f.Path)
			return true
		}
		return false
	})
	return pkg, len(pkg.GoFiles) > 0
}

// sourceFiles returns the source files a compile invocation was given.
//
// The go command always places the file list last and passes only ".go" files.
// No flag value can be mistaken for a source file: -importcfg, -embedcfg and
// -symabis all point at files named after the flag itself.
func sourceFiles(args []string) []fileArg {
	first := len(args)
	for first > 0 && strings.HasSuffix(args[first-1], ".go") {
		first--
	}
	files := make([]fileArg, 0, len(args)-first)
	for i := first; i < len(args); i++ {
		files = append(files, fileArg{Index: i, Path: args[i]})
	}
	return files
}

// skipFile applies the same exclusions the whole-tree instrumentor honours, so
// that the two workflows agree about what is covered.
func skipFile(cfg *Config, path string) bool {
	if cfg.SkipTestFiles && strings.HasSuffix(path, "_test.go") {
		return true
	}
	// Generated code has no hand-written branches worth attributing, and protobuf
	// output in particular is enormous.
	if strings.HasSuffix(path, ".pb.go") {
		return true
	}
	if cfg.Exclusions[path] {
		return true
	}
	for excluded := range cfg.Exclusions {
		excluded = strings.TrimSuffix(excluded, "/")
		if path == excluded || strings.HasPrefix(path, excluded+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func isSDKPackage(importPath string) bool {
	return importPath == common.ANTITHESIS_SDK_MODULE ||
		strings.HasPrefix(importPath, common.ANTITHESIS_SDK_MODULE+"/")
}

// inScope reports whether this package is one we were asked to instrument.
//
// With ANTITHESIS_INSTRUMENT set we match import path prefixes. With it unset the
// default is "code in the working tree": anything whose sources live outside
// GOROOT and outside the module cache. That covers the main module and any
// locally replaced module, and excludes downloaded dependencies.
func inScope(cfg *Config, importPath, sourcePath string) bool {
	if len(cfg.Instrument) > 0 {
		for _, prefix := range cfg.Instrument {
			prefix = strings.TrimSuffix(prefix, "/")
			if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
				return true
			}
		}
		return false
	}
	return isWorkingTreeSource(sourcePath)
}
