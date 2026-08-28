package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	qt "github.com/go-quicktest/qt"
)

func TestFlagValue(t *testing.T) {
	// Both spellings appear in real compile invocations: -importcfg takes a
	// separate argument, while -lang is written with an equals sign.
	args := strings.Fields("-o out.a -p app -lang=go1.24 -complete -importcfg /w/importcfg -pack a.go")

	qt.Assert(t, qt.Equals(flagValue(args, "-importcfg"), "/w/importcfg"))
	qt.Assert(t, qt.Equals(flagValue(args, "-lang"), "go1.24"))
	qt.Assert(t, qt.Equals(flagValue(args, "-p"), "app"))
	qt.Assert(t, qt.Equals(flagValue(args, "-embedcfg"), ""))
	// A boolean flag has no value, and must not swallow the next argument.
	qt.Assert(t, qt.Equals(flagValue(args, "-complete"), "-importcfg"))
}

func TestLanguageVersion(t *testing.T) {
	// go/types wants a "go1.N" string; the compiler is passed either form.
	qt.Assert(t, qt.Equals(languageVersion([]string{"-lang=go1.24"}), "go1.24"))
	qt.Assert(t, qt.Equals(languageVersion([]string{"-lang", "1.24"}), "go1.24"))
	// Absent -lang means "no constraint", which go/types spells as empty.
	qt.Assert(t, qt.Equals(languageVersion([]string{"-complete"}), ""))
}

func TestModuleRoot(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	qt.Assert(t, qt.IsNil(os.MkdirAll(nested, 0o755)))
	qt.Assert(t, qt.IsNil(os.WriteFile(filepath.Join(root, "go.mod"), []byte("module m\n"), 0o644)))

	// Found from the module root itself and from any depth beneath it.
	qt.Assert(t, qt.Equals(moduleRoot(root), root))
	qt.Assert(t, qt.Equals(moduleRoot(nested), root))
}

func TestModuleRootAbsentIsEmpty(t *testing.T) {
	// With no go.mod anywhere above, callers fall back to absolute paths rather
	// than inventing a root.
	dir := t.TempDir()
	if moduleRoot(dir) != "" {
		t.Skip("the temporary directory is inside a module; nothing to assert")
	}
	qt.Assert(t, qt.Equals(moduleRoot(dir), ""))
}

func TestCatalogFileMatchesTheWholeTreeName(t *testing.T) {
	// The two workflows must agree on the name, so that a catalog committed by
	// either one is recognised by the other.
	qt.Assert(t, qt.Equals(catalogFile, "antithesis_catalog.go"))
}
