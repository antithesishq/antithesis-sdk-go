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

	// Global logger
	LogWriter *common.LogWriter

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

	// Indicates that instrumentation is requested (true)
	// If set to (false) then perform assertion catalog scanning
	// without instrumentation, which is common
	// when execution is outside of the Antithesis environment
	WantsInstrumentor bool

	// Indicates that '*_test.go' files should be skipped (not instrumented)
	// default is false
	SkipTestFiles bool

	// Indicates that '*.pb.go' files should be skipped (not instrumented)
	// default is false
	SkipProtoBufFiles bool
}

func NewCommonConfig(args *args.Args) (*CommonConfig, error) {
	customerInputDirectory := common.GetAbsoluteDirectory(args.InputDir)

	if _, err := getModuleName(customerInputDirectory); err != nil {
		return nil, fmt.Errorf("unable to obtain go module name from %q", customerInputDirectory)
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
		WantsInstrumentor: args.WantsInstrumentor,
		LogWriter:         common.GetLogWriter(),
		SkipTestFiles:     args.SkipTestFiles,
		SkipProtoBufFiles: args.SkipProtoBufFiles,
	}, nil
}

// CommonConfig methods

func (cc *CommonConfig) GetSourceDir() string {
	return cc.InputDirectory
}

// Full instrumentation targets the customerDirectory
// Assertions only mode will target in-place (same as inputDirectory)
func (cc *CommonConfig) GetTargetDir() string {
	if cc.WantsInstrumentor {
		return cc.CustomerDirectory
	}
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

func getModuleName(inputDir string) (moduleName string, err error) {
	var moduleData []byte
	moduleName = ""
	moduleFilenamePath := filepath.Join(inputDir, "go.mod")
	if moduleData, err = os.ReadFile(moduleFilenamePath); err != nil {
		return
	}

	var f *modfile.File
	if f, err = modfile.ParseLax("go.mod", moduleData, nil); err == nil {
		moduleName = f.Module.Mod.Path
	}
	return
}
