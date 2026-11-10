package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/args"
	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
	"golang.org/x/mod/modfile"
)

// CommonConfig holds fields shared across both coverage instrumentation
// and assertion scanning.
type CommonConfig struct {
	// The set of exclusions that were obtained from
	// reading the 'excludeFile'. A map is used where
	// the value is always 'true', in lieu of a specific
	// 'set' abstraction not available for the version
	// of go used for this tool.
	Exclusions map[string]bool

	// SHA256 Hash (48-bits worth) of all the files
	// in sourceFiles
	FilesHash string

	// The base directry of a go module to be instrumented/cataloged
	// Contains a go.mod file
	InputDirectory string

	// Option file containing a list (one per line) of
	// any files or directories to be excluded from both
	// instumentation and assertion scanning.  Empty lines
	// and lines beginining with '#' are ignored.
	ExcludeFile string

	// created and written to during instrumentation.
	// Will contain a copy of the inputDirectory, where All
	// non-excluded '.go' files are instrumented
	CustomerDirectory string

	// All of the files (after exclusions) to be instrumented
	// and scanned for assertions that should appear in the
	// assertion catalog
	SourceFiles []string

	// Number of files skipped when creating the sourceFiles
	// list.
	FilesSkipped int

	// Indicates that '*_test.go' files should be skipped (not instrumented)
	// default is false
	SkipTestFiles bool

	// Indicates that '*.pb.go' files should be skipped (not instrumented)
	// default is false
	SkipProtoBufFiles bool

	// What a person calls the instrumented thing, written to the symbol
	// table's "# name" global: the go.mod module path unless -name overrides it.
	DisplayName string
}

func NewCommonConfig(args *args.Args) (*CommonConfig, error) {
	customerInputDirectory := common.GetAbsoluteDirectory(args.InputDir)

	modulePath, err := getModuleName(customerInputDirectory)
	if err != nil {
		return nil, fmt.Errorf("unable to obtain go module name from %q: %w", customerInputDirectory, err)
	}
	displayName := args.Name
	if displayName == "" {
		displayName = modulePath
	}

	customerDirectory := ""
	if args.WantsInstrumentor {
		outputDirectory := common.GetAbsoluteDirectory(args.OutputDir)
		customerDirectory = filepath.Join(outputDirectory, common.INSTRUMENTED_SOURCE_FOLDER)
	}

	return &CommonConfig{
		InputDirectory:    customerInputDirectory,
		CustomerDirectory: customerDirectory,
		ExcludeFile:       args.ExcludeFile,
		SkipTestFiles:     args.SkipTestFiles,
		SkipProtoBufFiles: args.SkipProtoBufFiles,
		DisplayName:       displayName,
	}, nil
}

// CommonConfig methods

func (cc *CommonConfig) GetSourceDir() string {
	return cc.InputDirectory
}

func (cc *CommonConfig) ParseExclusionsFile() (err error) {
	if cc.ExcludeFile == "" {
		return
	}
	cc.Exclusions = map[string]bool{}
	var parsedExclusions map[string]bool

	parsedExclusions, err = ParseExclusionsFile(cc.ExcludeFile, cc.InputDirectory)
	if err == nil {
		cc.Exclusions = parsedExclusions
	}
	return
}

// Private helpers

func getModuleName(inputDir string) (string, error) {
	moduleFilenamePath := filepath.Join(inputDir, "go.mod")
	moduleData, err := os.ReadFile(moduleFilenamePath)
	if err != nil {
		return "", err
	}
	f, err := modfile.ParseLax("go.mod", moduleData, nil)
	if err != nil {
		return "", err
	}
	if f.Module == nil {
		return "", fmt.Errorf("%s has no module directive", moduleFilenamePath)
	}
	return f.Module.Mod.Path, nil
}
