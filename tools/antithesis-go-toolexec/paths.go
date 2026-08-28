package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// The go command runs each toolchain program with the working directory set to
// the package's source directory, and it gives us no direct statement of GOROOT
// or GOMODCACHE
//
// goroot recovers GOROOT from the path of the tool we were asked to run, which is
// always $GOROOT/pkg/tool/<goos>_<goarch>/<tool>.
func goroot(toolPath string) string {
	dir := filepath.Dir(toolPath) // .../pkg/tool/linux_amd64
	dir = filepath.Dir(dir)       // .../pkg/tool
	dir = filepath.Dir(dir)       // .../pkg
	if root := filepath.Dir(dir); root != "." {
		return root
	}
	return runtime.GOROOT()
}

// goModCache resolves GOMODCACHE the same way the go command does.
func goModCache() string {
	if dir := strings.TrimSpace(os.Getenv("GOMODCACHE")); dir != "" {
		return dir
	}
	gopath := strings.TrimSpace(os.Getenv("GOPATH"))
	if gopath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		gopath = filepath.Join(home, "go")
	}
	// GOPATH is a list; only its first element is written to.
	if i := strings.IndexByte(gopath, os.PathListSeparator); i >= 0 {
		gopath = gopath[:i]
	}
	return filepath.Join(gopath, "pkg", "mod")
}

// isWorkingTreeSource reports whether path is source the user is working on, as
// opposed to a downloaded dependency or part of the standard library.
func isWorkingTreeSource(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	for _, external := range []string{goModCache(), goroot(os.Args[1])} {
		if external != "" && isUnder(abs, external) {
			return false
		}
	}
	return true
}

func isUnder(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// moduleRoot finds the directory holding the go.mod that governs the directory dir
//
// Returns "" when no go.mod is found, in which case callers fall back to
// absolute paths.
func moduleRoot(dir string) string {
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
