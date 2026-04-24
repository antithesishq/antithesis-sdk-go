package config

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/args"
	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
	commonconfig "github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/config"
)

// CoverageConfig holds fields specific to coverage instrumentation.
type CoverageConfig struct {
	// The modules inferred from scanning source files and
	// looking for files named go.mod
	// Each of these go.mod files may need the antithesis
	// module added as a dependency.
	dependentModules map[string]bool

	// The name of the symbol table file, which incorporates
	// the overall 'filesHash' and 'symbtablePrefix'
	SymbolTableFilename string

	// created and written during instrumentation.
	// Will contain the corresponding .tsv file expected
	// by the antithesis fuzzer
	SymbolsDirectory string

	// The instrumentation (only) base output directory
	// Is required to exist, and to be empty prior to instrumentation
	//
	// After instrumentation, will contain the subdirectories for
	// 'symbols' and 'customer'
	outputDirectory string

	// A prefix used to distinguish symbol table filenames
	// that will be used by the antithesis fuzzer.
	SymtablePrefix string

	// The version of SDK to use at runtime for CoverageInstrumentation
	instrumentorVersion string

	// The path to the SDK to use to create the notifier
	localSDKPath string

	// created and written to during instrumentation.
	// Will contain the antithesis notifier module (go.mod) and source (notifier.go)
	notifierDirectory string
}

func NewCoverageConfig(args *args.Args) (*CoverageConfig, error) {
	symtablePrefix := ""
	if args.SymPrefix != "" {
		symtablePrefix = args.SymPrefix + "-"
	}

	outputDirectory := ""
	symbolsDirectory := ""
	notifierDirectory := ""

	if args.WantsInstrumentor {
		customerInputDirectory := common.GetAbsoluteDirectory(args.InputDir)
		outputDirectory = common.GetAbsoluteDirectory(args.OutputDir)
		if err := common.ValidateDirectories(customerInputDirectory, outputDirectory); err != nil {
			return nil, err
		}

		customerDirectory := filepath.Join(outputDirectory, common.INSTRUMENTED_SOURCE_FOLDER)
		notifierDirectory = filepath.Join(outputDirectory, common.NOTIFIER_FOLDER)
		symbolsDirectory = filepath.Join(outputDirectory, common.SYMBOLS_FOLDER)
		if err := createOutputDirectories(customerDirectory, notifierDirectory, symbolsDirectory); err != nil {
			return nil, err
		}
	}

	return &CoverageConfig{
		outputDirectory:     outputDirectory,
		SymbolsDirectory:    symbolsDirectory,
		SymtablePrefix:      symtablePrefix,
		instrumentorVersion: args.InstrumentorVersion,
		localSDKPath:        args.LocalSDKPath,
		notifierDirectory:   notifierDirectory,
	}, nil
}

func (cov *CoverageConfig) GetSourceFiles(cc *commonconfig.CommonConfig) (sourceFiles []string, err error) {
	sourceFiles = []string{}
	cc.FilesSkipped = 0
	if err = cc.ParseExclusionsFile(); err != nil {
		return
	}

	numSkipped := 0
	if sourceFiles, numSkipped, err = cov.FindSourceCode(cc); err != nil {
		return
	}
	cc.FilesSkipped = numSkipped
	cc.FilesHash = common.HashFileContent(sourceFiles)
	return
}

// FindSourceCode scans an input directory recursively for .go files,
// skipping any files or directories specified in exclusions.
func (cov *CoverageConfig) FindSourceCode(cc *commonconfig.CommonConfig) (paths []string, numSkipped int, err error) {
	paths = []string{}
	numSkipped = 0

	cov.dependentModules = map[string]bool{}

	logger := common.Logger
	logger.Printf(common.Normal, "Scanning %s recursively for .go source", cc.InputDirectory)
	// Files are read in lexical order, i.e. we can later deterministically
	// hash their content: https://pkg.go.dev/path/filepath#WalkDir
	err = filepath.WalkDir(cc.InputDirectory,
		func(path string, info fs.DirEntry, erx error) error {
			if erx != nil {
				logger.Printf(common.Normal, "Error %v in directory %s; skipping", erx, path)
				return erx
			}

			if b := filepath.Base(path); strings.HasPrefix(b, ".") {
				logger.Printf(common.Debug, "Ignoring 'dot' directory: %s", path)
				if info.IsDir() {
					return fs.SkipDir
				}
				return nil
			}

			if b := filepath.Base(path); b == "testdata" {
				logger.Printf(common.Debug, "Ignoring 'testdata' directory: %s", path)
				if info.IsDir() {
					return fs.SkipDir
				}
				return nil
			}

			if cc.Exclusions[path] {
				if info.IsDir() {
					logger.Printf(common.Normal, "Ignoring excluded directory %s and its children", path)
					return fs.SkipDir
				}
				logger.Printf(common.Normal, "Skipping excluded file %s", path)
				numSkipped++
				return nil
			}
			if info.IsDir() {
				return nil
			}
			dir, baseFile := filepath.Split(path)
			ext := filepath.Ext(path)
			if ext != ".go" {
				if baseFile == "go.mod" {
					cov.dependentModules[filepath.Clean(dir)] = false
				}
				numSkipped++
				return nil
			}
			// This is the mandatory format of unit test file names.
			if cc.SkipTestFiles && strings.HasSuffix(baseFile, "_test.go") {
				logger.Printf(common.Info, "Skipping test file %s", path)
				numSkipped++
				return nil
			}

			if cc.SkipProtoBufFiles && strings.HasSuffix(baseFile, ".pb.go") {
				logger.Printf(common.Info, "Skipping generated file %s", path)
				numSkipped++
				return nil
			}

			paths = append(paths, path)

			return nil
		})
	if err != nil {
		err = fmt.Errorf("error walking input directory %s: %v", cc.InputDirectory, err)
	}

	return
}

func (cov *CoverageConfig) GetNotifierDirectory() string {
	return cov.notifierDirectory
}

func (cov *CoverageConfig) CreateNotifierModule(cc *commonconfig.CommonConfig) {
	notifierModuleName := common.NOTIFIER_MODULE_NAME

	if cc.WantsInstrumentor {
		common.NotifierDependencies(cov.notifierDirectory, notifierModuleName, cov.instrumentorVersion, cov.localSDKPath)
	}
}

func (cov *CoverageConfig) ShowDependentModules() {
	isText := ""
	common.Logger.Printf(common.Normal, "")
	common.Logger.Printf(common.Normal, "Module Usage Summary")
	for modName, used := range cov.dependentModules {
		isText = "is"
		if !used {
			isText = "is not"
		}
		common.Logger.Printf(common.Normal, "%s %s used", modName, isText)
	}
	common.Logger.Printf(common.Normal, "")
}

func (cov *CoverageConfig) UpdateDependentModules(file_name string) {
	ok := false
	isUsed := false
	this_dir := filepath.Clean(filepath.Dir(file_name))
	for !ok {
		common.Logger.Printf(common.Debug, "Checking if %q is a dependentModule", this_dir)
		if this_dir == "." {
			break
		}
		isUsed, ok = cov.dependentModules[this_dir]
		if ok {
			if !isUsed {
				cov.dependentModules[this_dir] = true
			}
			return
		} else {
			old_dir := this_dir
			this_dir = filepath.Clean(filepath.Dir(this_dir))
			ok = (old_dir == this_dir)
		}
	}
	common.Logger.Printf(common.Normal, "%q does not belong to a scanned module", file_name)
}

func (cov *CoverageConfig) WrapUp(cc *commonconfig.CommonConfig) {
	if !cc.WantsInstrumentor {
		return
	}

	notifierModule := common.FullNotifierName(cc.FilesHash)
	notifierRelPath := ".."
	localNotifier := filepath.Join(notifierRelPath, common.NOTIFIER_FOLDER)
	common.AddDependencies(cc.InputDirectory, cc.CustomerDirectory, cov.instrumentorVersion, notifierModule, localNotifier)

	someOffset := ""
	for modFolder, used := range cov.dependentModules {
		if used {
			someOffset = common.PathFromBaseDirectory(cc.InputDirectory, modFolder)
			if someOffset != "" {
				destModuleFolder := filepath.Join(cc.CustomerDirectory, someOffset)
				os.MkdirAll(destModuleFolder, 0777)

				basePath := filepath.Join(cc.CustomerDirectory, someOffset)
				targPath := cov.notifierDirectory
				if altDestModuleFolder, erx := filepath.Rel(basePath, targPath); erx == nil {
					common.AddDependencies(modFolder, destModuleFolder, cov.instrumentorVersion, notifierModule, altDestModuleFolder)
				}
			}
		}
	}

	if err := common.CopyRecursiveDir(cc.InputDirectory, cc.CustomerDirectory); err == nil {
		common.Logger.Printf(common.Normal, "All other files copied unmodified from %s to %s", cc.InputDirectory, cc.CustomerDirectory)
	} else {
		common.Logger.Printf(common.Normal, "CopyRecursiveDir err: %s", err.Error())
	}

	if common.Logger.VerboseLevel(common.Info) {
		common.ShowDirRecursive(cc.CustomerDirectory, "instrumented files")
	}

	if cov.localSDKPath == "" {
		common.FetchDependencies(cc.CustomerDirectory)
		common.Logger.Printf(common.Normal, "Downloaded Antithesis dependencies")
	}
}

// Private helpers

func createOutputDirectories(customerDirectory, notifierDirectory, symbolsDirectory string) (err error) {
	if err = os.Mkdir(customerDirectory, 0755); err != nil {
		return
	}
	if err = os.Mkdir(notifierDirectory, 0755); err != nil {
		return
	}
	err = os.Mkdir(symbolsDirectory, 0755)
	return
}
