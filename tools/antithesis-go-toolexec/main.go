// antithesis-go-toolexec instruments Go code for coverage and catalogs Antithesis assertions
// from inside the build as a `go build -toolexec` wrapper.
//
// Usage:
//
//	go build -toolexec=antithesis-go-toolexec ./...
//
// or, so that no build command has to change:
//
//	export GOFLAGS=-toolexec=antithesis-go-toolexec
//
// The go command runs this program in place of each toolchain program, passing
// the real program and its arguments. We pass almost everything through
// untouched; see compile.go for the three edits we make to a compile invocation.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
)

func main() { os.Exit(run()) }

// run does the work and returns an exit status, so that deferred cleanup runs.
func run() int {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		return 2
	}
	if os.Args[1] == "-h" || os.Args[1] == "-help" || os.Args[1] == "--help" {
		fmt.Println(usage)
		return 0
	}

	cfg, err := loadConfig()
	if err != nil {
		return errorf("%s", err)
	}
	// The shared instrumentor code logs at Normal, which prints at the default
	// verbosity. Anything written to stderr while a compile action runs stops the
	// go command from caching that action, so unless verbosity was actually asked
	// for, sit below Normal and stay silent.
	verbosity := cfg.Verbosity
	if verbosity <= common.Normal {
		verbosity = common.Normal - 1
	}
	common.NewLogWriter("", verbosity)

	realTool, args := os.Args[1], os.Args[2:]
	tool := toolName(realTool)

	// The go command asks each tool to identify itself and folds the answer into
	// the action cache key (cmd/go/internal/work/buildid.go). Appending to that
	// line is what keeps instrumented objects in different cache entries from
	// uninstrumented ones -- without it the build silently reuses stock objects.
	if isVersionProbe(args) {
		out, err := exec.Command(realTool, args...).Output()
		if err != nil {
			return exitStatus(err)
		}
		suffix, err := cfg.ToolIDSuffix()
		if err != nil {
			// Without a trustworthy identity the cache key would be wrong, and a
			// wrong cache key is silent. Refuse instead.
			return errorf("%s", err)
		}
		fmt.Printf("%s %s\n", strings.TrimSpace(string(out)), suffix)
		return 0
	}

	switch tool {
	case "compile":
		if pkg, ok := selectedPackage(cfg, args); ok {
			var cleanup func()
			if args, cleanup, err = rewriteCompile(cfg, pkg, args); err != nil {
				return errorf("instrumenting %s: %s", pkg.ImportPath, err)
			}
			// The generated files are needed only until the compiler has read
			// them, and one build can compile thousands of packages.
			defer cleanup()
		}
	case "link":
		if args, err = prepareLink(cfg, args); err != nil {
			return errorf("preparing link: %s", err)
		}
	}

	cmd := exec.Command(realTool, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return exitStatus(err)
	}
	return 0
}

// toolName reduces the program path the go command handed us to a bare tool
// name, so "/…/pkg/tool/linux_amd64/compile" becomes "compile". Anything we do
// not recognise (notably the C compiler during a cgo build, which is invoked as
// `clang -### -x c -c -`) falls through untouched.
func toolName(path string) string {
	return strings.TrimSuffix(filepath.Base(path), ".exe")
}

// isVersionProbe reports whether this invocation is the go command asking the
// tool for its build ID rather than asking it to do work. The Go tools use
// -V=full; the C compiler is probed with -### instead, and that output is hashed
// rather than parsed, so we leave it alone.
func isVersionProbe(args []string) bool {
	return len(args) > 0 && args[len(args)-1] == "-V=full"
}

// exitStatus reports the wrapped tool's own status. Collapsing every failure to 1
// would make the go command misreport which step failed and why.
func exitStatus(err error) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return errorf("%s", err)
}

// errorf reports a failure of the wrapper itself. It goes to stderr rather than
// through common.Logger, because the shared logger prefixes its own file and line
// -- misleading in a build log -- and because these must be seen at any verbosity.
func errorf(format string, args ...any) int {
	fmt.Fprintf(os.Stderr, toolName(os.Args[0])+": "+format+"\n", args...)
	return 1
}

// progressf reports what the wrapper did, on the same terms as errorf.
func progressf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, toolName(os.Args[0])+": "+format+"\n", args...)
}

const usage = `antithesis-go-toolexec is a "go build -toolexec" wrapper that adds
Antithesis coverage instrumentation during the build.

  go build -toolexec=antithesis-go-toolexec ./...
  GOFLAGS=-toolexec=antithesis-go-toolexec go build ./...

It is configured by environment variables, because the go command controls its
argument list:

  ANTITHESIS_INSTRUMENT       comma-separated import path prefixes to instrument.
                              Defaults to the main module of the build.
  ANTITHESIS_SYMBOLS_DIR      where symbol tables are collected. Default /symbols.
  ANTITHESIS_EXCLUDE          path to an exclusions file, as accepted by
                              antithesis-go-instrumentor -exclude.
  ANTITHESIS_SYMBOL_PREFIX    prefix for symbol table file names.
  ANTITHESIS_SDK_MODULE_DIR   directory of an antithesis-sdk-go checkout, used to
                              supply the SDK when the module being built does not
                              depend on it.
  ANTITHESIS_SKIP_TEST_FILES  set to 1 to leave _test.go files uninstrumented.
  ANTITHESIS_SKIP_CATALOG     set to 1 to skip assertion cataloguing.
  ANTITHESIS_SKIP_COVERAGE    set to 1 to skip coverage instrumentation. With
                              cataloguing left on, this is the equivalent of
                              antithesis-go-instrumentor -assert_only.
  ANTITHESIS_VERBOSE          0-3; logs to stderr during the build. Note that a
                              compile that writes to stderr is not cached, so
                              this effectively disables incremental builds.

Assertion catalogs are generated automatically, one per instrumented package, so
no separate antithesis-go-instrumentor pass is required.`
