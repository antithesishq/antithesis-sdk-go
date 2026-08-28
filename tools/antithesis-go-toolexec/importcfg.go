package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// An importcfg tells the compiler and linker where each import path's compiled
// archive lives. The go command writes one per action, listing only the packages
// it computed as that action's dependencies. An import we inject is not in
// there, and the compiler cannot resolve it until we add a line.

// importCfg is a parsed importcfg file, preserving the original text so that
// anything we do not understand survives a rewrite.
type importCfg struct {
	text string
	// archives maps each resolvable import path to the compiled archive
	// implementing it. Cataloguing type-checks against these.
	archives map[string]string
}

func readImportCfg(path string) (*importCfg, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading importcfg %s: %w", path, err)
	}
	cfg := &importCfg{text: string(body), archives: map[string]string{}}
	scanner := bufio.NewScanner(strings.NewReader(cfg.text))
	for scanner.Scan() {
		rest, ok := strings.CutPrefix(strings.TrimSpace(scanner.Text()), "packagefile ")
		if !ok {
			continue
		}
		if importPath, archive, found := strings.Cut(rest, "="); found {
			cfg.archives[importPath] = archive
		}
	}
	return cfg, scanner.Err()
}

// Has reports whether the compiler or linker can already resolve importPath.
func (c *importCfg) Has(importPath string) bool { _, ok := c.archives[importPath]; return ok }

// withEntries writes a copy of the importcfg extended with entries for packages
// it does not already list. Existing entries always win; a package the go command
// built for this action must never be shadowed by one we resolved separately.
func (c *importCfg) withEntries(dir, name string, entries []packageEntry) (string, error) {
	var added strings.Builder
	for _, entry := range entries {
		if c.Has(entry.ImportPath) {
			continue
		}
		fmt.Fprintf(&added, "packagefile %s=%s\n", entry.ImportPath, entry.ArchiveFile)
	}
	if added.Len() == 0 {
		return "", nil
	}

	text := c.text
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(text+added.String()), 0o644); err != nil {
		return "", fmt.Errorf("writing patched importcfg: %w", err)
	}
	return path, nil
}

// patchImportCfg replaces the importcfg at args[valueIndex] with a copy able to
// resolve entries, if any are missing. It returns args unchanged when the
// existing file already suffices.
func patchImportCfg(args []string, valueIndex int, entries []packageEntry) ([]string, error) {
	original := args[valueIndex]
	cfg, err := readImportCfg(original)
	if err != nil {
		return nil, err
	}

	work, err := os.MkdirTemp("", "antithesis-importcfg-")
	if err != nil {
		return nil, err
	}
	patched, err := cfg.withEntries(work, filepath.Base(original), entries)
	if err != nil {
		return nil, err
	}
	if patched == "" {
		os.Remove(work)
		return args, nil
	}
	args[valueIndex] = patched
	return args, nil
}
