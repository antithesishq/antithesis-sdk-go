package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
)

// Symbol tables are written under GOCACHE rather than straight into the image's
// symbols directory. This ensures that a cache hit still has its symbol table since
// the two are only ever discarded together.

const shardDirName = "antithesis-symbols"

// indexSuffix marks the files recording which symbol table belongs to which
// package, so the link step can report a gap instead of silently shipping one.
const indexSuffix = ".pkg"

// shardDirectory returns the per-build-cache directory holding symbol tables.
func shardDirectory() (string, error) {
	cache := strings.TrimSpace(os.Getenv("GOCACHE"))
	if cache == "" {
		// GOCACHE is set for every tool invocation the go command makes, so an
		// empty value means we are being run some other way.
		return "", errors.New("GOCACHE is not set; this tool must be run via 'go build -toolexec'")
	}
	dir := filepath.Join(cache, shardDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating %s: %w", dir, err)
	}
	return dir, nil
}

// recordShard notes that importPath's edges are described by symbolName.
func recordShard(shardDir, importPath, symbolName string) error {
	path := filepath.Join(shardDir, escapeImportPath(importPath)+indexSuffix)
	if err := os.WriteFile(path, []byte(symbolName+"\n"), 0o644); err != nil {
		return fmt.Errorf("recording symbol table for %s: %w", importPath, err)
	}
	return nil
}

// instrumentedImportPaths lists every package known to have been instrumented into
// this build cache. The link step uses it to tell an uninstrumented build (nothing
// to do) from an instrumented one (the SDK must be linkable).
func instrumentedImportPaths(cfg *Config) []string {
	shardDir, err := shardDirectory()
	if err != nil {
		return nil
	}
	entries, err := os.ReadDir(shardDir)
	if err != nil {
		return nil
	}
	var paths []string
	for _, entry := range entries {
		if name := entry.Name(); strings.HasSuffix(name, indexSuffix) {
			paths = append(paths, unescapeImportPath(strings.TrimSuffix(name, indexSuffix)))
		}
	}
	return paths
}

// collectShards copies symbol tables into the directory the image build harvests.
//
// It runs at link time because that is the only step guaranteed to happen on a
// build that produces a binary. Copying everything present, rather than only what
// this binary references, means the destination can be a superset.
func collectShards(cfg *Config) error {
	shardDir, err := shardDirectory()
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(shardDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	available := map[string]bool{}
	var indexed []string
	for _, entry := range entries {
		switch name := entry.Name(); {
		case strings.HasSuffix(name, common.SYMBOLS_FILE_SUFFIX):
			available[name] = true
		case strings.HasSuffix(name, indexSuffix):
			indexed = append(indexed, name)
		}
	}
	if len(available) == 0 && len(indexed) == 0 {
		return nil
	}

	// A recorded package with no symbol table means someone removed files from
	// under us. Say so rather than shipping a binary we cannot attribute.
	var missing []string
	for _, index := range indexed {
		body, err := os.ReadFile(filepath.Join(shardDir, index))
		if err != nil {
			return err
		}
		symbolName := strings.TrimSpace(string(body))
		if symbolName != "" && !available[symbolName] {
			missing = append(missing, fmt.Sprintf("%s (%s)",
				unescapeImportPath(strings.TrimSuffix(index, indexSuffix)), symbolName))
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("symbol tables missing from %s for %d instrumented %s: %s\n"+
			"Remove the build cache (go clean -cache) and rebuild",
			shardDir, len(missing), common.Pluralize(len(missing), "package"), strings.Join(missing, ", "))
	}

	if err := os.MkdirAll(cfg.SymbolsDir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", cfg.SymbolsDir, err)
	}
	copied := 0
	for name := range available {
		body, err := os.ReadFile(filepath.Join(shardDir, name))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(cfg.SymbolsDir, name), body, 0o644); err != nil {
			return fmt.Errorf("copying %s to %s: %w", name, cfg.SymbolsDir, err)
		}
		copied++
	}
	progressf("collected %d symbol %s into %s",
		copied, common.Pluralize(copied, "table"), cfg.SymbolsDir)
	return nil
}

// escapeImportPath makes an import path usable as a file name. Import paths may
// contain '/' and, for case-insensitive filesystems, upper-case letters that the
// module cache would encode; here a single reversible substitution is enough.
func escapeImportPath(importPath string) string {
	return strings.ReplaceAll(importPath, "/", "%")
}

func unescapeImportPath(name string) string {
	return strings.ReplaceAll(name, "%", "/")
}
