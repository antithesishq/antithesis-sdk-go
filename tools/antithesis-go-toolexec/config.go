package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/antithesishq/antithesis-sdk-go/internal"
	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/config"
)

// sdkInstrumentationPackage is the package supplying RegisterModule. Instrumented
// packages get an import of it and a package-level variable holding the handle.
const sdkInstrumentationPackage = "github.com/antithesishq/antithesis-sdk-go/instrumentation"

// symbolsDirDefault is where the container image build looks for symbol tables;
// see environment/container_module/symbols.nix.
const symbolsDirDefault = "/symbols"

// Config is everything the wrapper needs, gathered from the environment. The go
// command owns our argument list, so there are no flags of our own.
type Config struct {
	// Instrument lists import path prefixes to instrument. Empty means "the main
	// module of this build", resolved lazily from the build itself.
	Instrument []string

	// SymbolsDir is where symbol tables are collected for the image.
	SymbolsDir string

	// SDKModuleDir optionally points at an antithesis-sdk-go checkout, used to
	// supply the SDK to modules that do not depend on it.
	SDKModuleDir string

	// SymbolPrefix is prepended to symbol table file names.
	SymbolPrefix string

	// Exclusions are paths the instrumentor must leave alone.
	Exclusions map[string]bool

	// Coverage and Catalog select the features to apply. They are independent:
	// turning coverage off while leaving cataloguing on is the equivalent of
	// antithesis-go-instrumentor's -assert_only.
	Coverage bool
	Catalog  bool

	SkipTestFiles bool
	Verbosity     common.Verbosity
}

func loadConfig() (*Config, error) {
	cfg := &Config{
		Instrument:    splitList(os.Getenv("ANTITHESIS_INSTRUMENT")),
		SymbolsDir:    envOr("ANTITHESIS_SYMBOLS_DIR", symbolsDirDefault),
		SDKModuleDir:  strings.TrimSpace(os.Getenv("ANTITHESIS_SDK_MODULE_DIR")),
		SymbolPrefix:  strings.TrimSpace(os.Getenv("ANTITHESIS_SYMBOL_PREFIX")),
		Coverage:      !isTrue(os.Getenv("ANTITHESIS_SKIP_COVERAGE")),
		Catalog:       !isTrue(os.Getenv("ANTITHESIS_SKIP_CATALOG")),
		SkipTestFiles: isTrue(os.Getenv("ANTITHESIS_SKIP_TEST_FILES")),
	}

	if v := strings.TrimSpace(os.Getenv("ANTITHESIS_VERBOSE")); v != "" {
		level, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("ANTITHESIS_VERBOSE=%q is not a number", v)
		}
		cfg.Verbosity = common.Verbosity(level)
	}

	if excludeFile := strings.TrimSpace(os.Getenv("ANTITHESIS_EXCLUDE")); excludeFile != "" {
		// Exclusions are recorded relative to the directory holding the file, the
		// same convention antithesis-go-instrumentor -exclude uses.
		base := filepath.Dir(excludeFile)
		exclusions, err := config.ParseExclusionsFile(excludeFile, base)
		if err != nil {
			return nil, fmt.Errorf("reading ANTITHESIS_EXCLUDE=%q: %w", excludeFile, err)
		}
		cfg.Exclusions = exclusions
	}

	if cfg.SDKModuleDir != "" {
		abs, err := filepath.Abs(cfg.SDKModuleDir)
		if err != nil {
			return nil, fmt.Errorf("resolving ANTITHESIS_SDK_MODULE_DIR: %w", err)
		}
		cfg.SDKModuleDir = abs
	}

	return cfg, nil
}

// ToolIDSuffix is appended to each tool's -V=full output, becoming part of the
// go command's action cache key. It must change whenever anything that affects
// the bytes we emit changes, or the build will serve stale objects.
//
// The format is deliberately close to the toolchain's own " X:experiment" suffix.
// cmd/go requires the line to keep the shape "<tool> version <version> ..." and
// only ever appends to it, so extending it here is safe.
func (c *Config) ToolIDSuffix() (string, error) {
	identity, err := instrumentorIdentity()
	if err != nil {
		return "", err
	}
	inputs := []string{
		"sdk=" + internal.SDK_Version,
		"instrumentor=" + identity,
		"prefix=" + c.SymbolPrefix,
		"instrument=" + strings.Join(c.Instrument, ","),
		"skiptests=" + strconv.FormatBool(c.SkipTestFiles),
		"coverage=" + strconv.FormatBool(c.Coverage),
		"catalog=" + strconv.FormatBool(c.Catalog),
	}
	// Exclusions change which files get edges, so they belong in the key.
	excluded := make([]string, 0, len(c.Exclusions))
	for path := range c.Exclusions {
		excluded = append(excluded, path)
	}
	sort.Strings(excluded)
	inputs = append(inputs, "exclude="+strings.Join(excluded, ","))

	sum := sha256.Sum256([]byte(strings.Join(inputs, "\x00")))
	return "antithesis:" + internal.SDK_Version + "+" + hex.EncodeToString(sum[:])[:16], nil
}

func envOr(name, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return fallback
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func isTrue(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "t", "true", "y", "yes":
		return true
	}
	return false
}
