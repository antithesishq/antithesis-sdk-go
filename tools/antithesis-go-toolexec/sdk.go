package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
)

// packageEntry is one resolvable package: an import path and the compiled archive
// implementing it.
type packageEntry struct {
	ImportPath  string
	ArchiveFile string
}

// recursionGuard stops the nested `go list` below from re-entering this wrapper.
// Without it, resolving the SDK would instrument the SDK, which would need the
// SDK resolved, and so on.
const recursionGuard = "ANTITHESIS_TOOLEXEC_RESOLVING"

// ensureSDKImportable makes the SDK's instrumentation package resolvable to the
// compile invocation, patching -importcfg only if it is not already listed.
func ensureSDKImportable(cfg *Config, args []string, importCfgIndex int) ([]string, error) {
	existing, err := readImportCfg(args[importCfgIndex])
	if err != nil {
		return nil, err
	}
	if existing.Has(sdkInstrumentationPackage) {
		return args, nil
	}

	entries, err := resolveSDK(cfg)
	if err != nil {
		return nil, err
	}
	return patchImportCfg(args, importCfgIndex, entries)
}

// prepareLink gives the linker the same entries. The linker resolves every import
// path in the binary, including ones only reachable through our generated
// registration files, and it runs on any build that has to produce a binary --
// even one where every package was a cache hit and this wrapper never instrumented
// anything.
func prepareLink(cfg *Config, args []string) ([]string, error) {
	importCfgIndex := -1
	for i, arg := range args {
		if arg == "-importcfg" && i+1 < len(args) {
			importCfgIndex = i + 1
		}
	}
	if importCfgIndex < 0 {
		return args, nil
	}

	// With coverage off there are no symbol tables to collect and nothing that
	// references the instrumentation package, so the link needs no help. Catalogs
	// call into the assert package, which any package containing an assertion
	// already imports.
	if !cfg.Coverage {
		return args, nil
	}

	if err := collectShards(cfg); err != nil {
		return nil, err
	}

	existing, err := readImportCfg(args[importCfgIndex])
	if err != nil {
		return nil, err
	}
	if existing.Has(sdkInstrumentationPackage) {
		return args, nil
	}
	// Nothing in this binary references the SDK through the go command's own
	// dependency graph. If we instrumented nothing, that is simply an
	// uninstrumented build and there is nothing to add.
	if !existing.hasAnyOf(instrumentedImportPaths(cfg)) {
		return args, nil
	}

	entries, err := resolveSDK(cfg)
	if err != nil {
		return nil, err
	}
	return patchImportCfg(args, importCfgIndex, entries)
}

// hasAnyOf reports whether the importcfg lists any of these packages.
func (c *importCfg) hasAnyOf(importPaths []string) bool {
	return slices.ContainsFunc(importPaths, c.Has)
}

// resolveSDK returns the SDK's instrumentation package and everything it needs,
// as packagefile entries.
//
// The archives come from `go list -deps -export`, which builds each package and
// reports where the build cache put it. Two sources are possible:
//
//   - the module being built, when it already requires the SDK; or
//   - a separate SDK checkout named by ANTITHESIS_SDK_MODULE_DIR, which lets a
//     module with no Antithesis dependency at all be instrumented without editing
//     its go.mod.
//
// The second case has a caveat worth understanding. Those archives are built
// outside the main build, so any setting affecting object compatibility --
// GOARCH, -race, the toolchain, the active GOEXPERIMENT set -- must match. We
// inherit the environment, which covers most of it, and a mismatch fails loudly
// at link time with a fingerprint error rather than producing a bad binary.
func resolveSDK(cfg *Config) ([]packageEntry, error) {
	if os.Getenv(recursionGuard) != "" {
		return nil, fmt.Errorf("refusing to resolve %s recursively", sdkInstrumentationPackage)
	}

	command := exec.Command("go", "list", "-deps", "-export", "-f",
		"{{if .Export}}{{.ImportPath}}\t{{.Export}}{{end}}", sdkInstrumentationPackage)
	command.Env = append(os.Environ(),
		recursionGuard+"=1",
		// Our own -toolexec must not apply to this build.
		"GOFLAGS="+withoutToolexec(os.Getenv("GOFLAGS")),
	)
	if cfg.SDKModuleDir != "" {
		command.Dir = cfg.SDKModuleDir
	}

	output, err := command.Output()
	if err != nil {
		return nil, sdkResolutionError(cfg, err)
	}

	var entries []packageEntry
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		importPath, archive, found := strings.Cut(strings.TrimSpace(scanner.Text()), "\t")
		if !found || importPath == "" || archive == "" {
			continue
		}
		entries = append(entries, packageEntry{ImportPath: importPath, ArchiveFile: archive})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("resolved no archives for %s", sdkInstrumentationPackage)
	}
	common.Logger.Printf(common.Debug, "Resolved %d SDK %s", len(entries), common.Pluralize(len(entries), "package"))
	return entries, nil
}

func sdkResolutionError(cfg *Config, err error) error {
	var stderr string
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		stderr = strings.TrimSpace(string(exitErr.Stderr))
	}
	if cfg.SDKModuleDir != "" {
		return fmt.Errorf("could not build %s from ANTITHESIS_SDK_MODULE_DIR=%s: %w\n%s",
			sdkInstrumentationPackage, cfg.SDKModuleDir, err, stderr)
	}
	return fmt.Errorf(`could not resolve %s: %w
%s
The module being built does not require the Antithesis SDK. Either add it:

    go get %s

or point the wrapper at an SDK checkout:

    export ANTITHESIS_SDK_MODULE_DIR=/path/to/antithesis-sdk-go`,
		sdkInstrumentationPackage, err, stderr, common.ANTITHESIS_SDK_MODULE)
}

// withoutToolexec strips -toolexec from a GOFLAGS value so a nested go command
// does not re-enter this wrapper. Setting GOFLAGS explicitly is not enough on its
// own, because the value we inherit may well contain it.
func withoutToolexec(goflags string) string {
	kept := make([]string, 0, 4)
	for _, flag := range strings.Fields(goflags) {
		if flag == "-toolexec" || strings.HasPrefix(flag, "-toolexec=") {
			continue
		}
		kept = append(kept, flag)
	}
	return strings.Join(kept, " ")
}
