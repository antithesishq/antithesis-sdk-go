package config

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/args"
	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
	"golang.org/x/mod/modfile"
)

// Capitalized struct items are accessed outside this file
type Config struct {
	// The set of exclusions that were obtained from
	// reading the 'excludeFile'. A map is used where
	// the value is always 'true', in lieu of a specific
	// 'set' abstraction not available for the version
	// of go used for this tool.
	exclusions map[string]bool

	// The modules inferred from scanning source files and
	// looking for files named go.mod
	// Each of these go.mod files may need the antithesis
	// module added as a dependency.
	dependentModules map[string]bool

	// Global logger
	logWriter *common.LogWriter

	// The name of the symbol table file, which incorporates
	// the overall 'filesHash' and 'symbtablePrefix'
	SymbolTableFilename string

	// SHA256 Hash (48-bits worth) of all the files
	// in sourceFiles
	FilesHash string

	// created and written during instrumentation.
	// Will contain the corresponding .tsv file expected
	// by the antithesis fuzzer
	SymbolsDirectory string

	// The base directory where assertion catalog(s) will be written.
	// For assert-only mode, this is the inputDirectory.
	// For full instrumentation, this is the customerDirectory.
	// Per-binary catalogs are placed in subdirectories matching
	// each main package's relative position.
	catalogBaseDir string

	// The instrumentation (only) base output directory
	// Is required to exist, and to be empty prior to instrumentation
	//
	// After instrumentation, will contain the subdirectories for
	// 'symbols' and 'customer'
	outputDirectory string

	// A prefix used to distinguish symbol table filenames
	// that will be used by the antithesis fuzzer.
	SymtablePrefix string

	// The base directry of a go module to be instrumented/cataloged
	// Contains a go.mod file
	InputDirectory string

	// Option file containing a list (one per line) of
	// any files or directories to be excluded from both
	// instumentation and assertion scanning.  Empty lines
	// and lines beginining with '#' are ignored.
	excludeFile string

	// created and written to during instrumentation.
	// Will contain a copy of the inputDirectory, where All
	// non-excluded '.go' files are instrumented
	CustomerDirectory string

	// The version of SDK to use at runtime for CoverageInstrumentation
	instrumentorVersion string

	// The path to the SDK to use to create the notifier
	localSDKPath string

	// created and written to during instrumentation.
	// Will contain the antithesis notifier module (go.mod) and source (notifier.go)
	notifierDirectory string

	// All of the files (after exclusions) to be instrumented
	// and scanned for assertions that should appear in the
	// assertion catalog
	sourceFiles []string

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
	skipTestFiles bool

	// Indicates that '*.pb.go' files should be skipped (not instrumented)
	// default is false
	skipProtoBufFiles bool
}

func NewConfig(args *args.Args) (cfg *Config, err error) {
	outputDirectory := ""
	customerInputDirectory := common.GetAbsoluteDirectory(args.InputDir)
	if args.WantsInstrumentor {
		outputDirectory = common.GetAbsoluteDirectory(args.OutputDir)
		err = common.ValidateDirectories(customerInputDirectory, outputDirectory)
	}

	symtablePrefix := ""
	if args.SymPrefix != "" {
		symtablePrefix = args.SymPrefix + "-"
	}

	if err == nil {
		if _, err = getModuleName(customerInputDirectory); err != nil {
			err = fmt.Errorf("unable to obtain go module name from %q", customerInputDirectory)
		}
	}

	customerDirectory := ""
	notifierDirectory := ""
	symbolsDirectory := ""
	if args.WantsInstrumentor {
		customerDirectory = filepath.Join(outputDirectory, common.INSTRUMENTED_SOURCE_FOLDER)
		notifierDirectory = filepath.Join(outputDirectory, common.NOTIFIER_FOLDER)
		symbolsDirectory = filepath.Join(outputDirectory, common.SYMBOLS_FOLDER)
		if err == nil {
			err = createOutputDirectories(customerDirectory, notifierDirectory, symbolsDirectory)
		}
	}

	if err != nil {
		return
	}

	catalogBaseDir := customerInputDirectory
	if args.WantsInstrumentor {
		catalogBaseDir = customerDirectory
	}

	cfg = &Config{
		outputDirectory:     outputDirectory,
		InputDirectory:      customerInputDirectory,
		CustomerDirectory:   customerDirectory,
		notifierDirectory:   notifierDirectory,
		SymbolsDirectory:    symbolsDirectory,
		catalogBaseDir:      catalogBaseDir,
		excludeFile:         args.ExcludeFile,
		WantsInstrumentor:   args.WantsInstrumentor,
		SymtablePrefix:      symtablePrefix,
		instrumentorVersion: args.InstrumentorVersion,
		localSDKPath:        args.LocalSDKPath,
		logWriter:           common.GetLogWriter(),
		skipTestFiles:       args.SkipTestFiles,
		skipProtoBufFiles:   args.SkipProtoBufFiles,
	}
	return
}

func (cfg *Config) GetSourceFiles() (sourceFiles []string, err error) {
	sourceFiles = []string{}
	cfg.FilesSkipped = 0
	if err = cfg.ParseExclusionsFile(); err != nil {
		return
	}

	numSkipped := 0
	if sourceFiles, numSkipped, err = cfg.FindSourceCode(); err != nil {
		return
	}
	cfg.FilesSkipped = numSkipped
	cfg.FilesHash = common.HashFileContent(sourceFiles)
	return
}


func (cfg *Config) WrapUp() {
	if !cfg.WantsInstrumentor {
		return
	}

	notifierModule := common.FullNotifierName(cfg.FilesHash)
	notifierRelPath := ".."
	localNotifier := filepath.Join(notifierRelPath, common.NOTIFIER_FOLDER)
	common.AddDependencies(cfg.InputDirectory, cfg.CustomerDirectory, cfg.instrumentorVersion, notifierModule, localNotifier)

	someOffset := ""
	for modFolder, used := range cfg.dependentModules {
		if used {
			someOffset = common.PathFromBaseDirectory(cfg.InputDirectory, modFolder)
			if someOffset != "" {
				destModuleFolder := filepath.Join(cfg.CustomerDirectory, someOffset)
				os.MkdirAll(destModuleFolder, 0777)

				basePath := filepath.Join(cfg.CustomerDirectory, someOffset)
				targPath := cfg.notifierDirectory
				if altDestModuleFolder, erx := filepath.Rel(basePath, targPath); erx == nil {
					common.AddDependencies(modFolder, destModuleFolder, cfg.instrumentorVersion, notifierModule, altDestModuleFolder)
				}
			}
		}
	}

	if err := common.CopyRecursiveDir(cfg.InputDirectory, cfg.CustomerDirectory); err == nil {
		cfg.logWriter.Printf("All other files copied unmodified from %s to %s", cfg.InputDirectory, cfg.CustomerDirectory)
	} else {
		cfg.logWriter.Printf("CopyRecursiveDir err: %s", err.Error())
	}

	if cfg.logWriter.VerboseLevel(1) {
		common.ShowDirRecursive(cfg.CustomerDirectory, "instrumented files")
	}

	if cfg.localSDKPath == "" {
		common.FetchDependencies(cfg.CustomerDirectory)
		cfg.logWriter.Printf("Downloaded Antithesis dependencies")
	}
}

func (cfg *Config) GetSourceDir() string {
	return cfg.InputDirectory
}

// Full instrumentation targets the customerDirectory
// Assertions only mode will target in-place (same as inputDirectory)
func (cfg *Config) GetTargetDir() string {
	if cfg.WantsInstrumentor {
		return cfg.CustomerDirectory
	}
	return cfg.InputDirectory
}


func (cfg *Config) CreateNotifierModule() {
	notifierModuleName := common.NOTIFIER_MODULE_NAME

	if cfg.WantsInstrumentor {
		common.NotifierDependencies(cfg.notifierDirectory, notifierModuleName, cfg.instrumentorVersion, cfg.localSDKPath)
	}
}

func (cfg *Config) ParseExclusionsFile() (err error) {
	if cfg.excludeFile == "" {
		return
	}
	cfg.exclusions = map[string]bool{}
	var parsedExclusions map[string]bool

	parsedExclusions, err = ParseExclusionsFile(cfg.excludeFile, cfg.InputDirectory)
	if err == nil {
		cfg.exclusions = parsedExclusions
	}
	return
}

// FindSourceCode scans an input directory recursively for .go files,
// skipping any files or directories specified in exclusions.
func (cfg *Config) FindSourceCode() (paths []string, numSkipped int, err error) {
	paths = []string{}
	numSkipped = 0

	cfg.dependentModules = map[string]bool{}

	cfg.logWriter.Printf("Scanning %s recursively for .go source", cfg.InputDirectory)
	// Files are read in lexical order, i.e. we can later deterministically
	// hash their content: https://pkg.go.dev/path/filepath#WalkDir
	err = filepath.WalkDir(cfg.InputDirectory,
		func(path string, info fs.DirEntry, erx error) error {
			if erx != nil {
				cfg.logWriter.Printf("Error %v in directory %s; skipping", erx, path)
				return erx
			}

			if b := filepath.Base(path); strings.HasPrefix(b, ".") {
				if cfg.logWriter.VerboseLevel(2) {
					cfg.logWriter.Printf("Ignoring 'dot' directory: %s", path)
				}
				if info.IsDir() {
					return fs.SkipDir
				}
				return nil
			}

			if b := filepath.Base(path); b == "testdata" {
				if cfg.logWriter.VerboseLevel(2) {
					cfg.logWriter.Printf("Ignoring 'testdata' directory: %s", path)
				}
				if info.IsDir() {
					return fs.SkipDir
				}
				return nil
			}

			if cfg.exclusions[path] {
				if info.IsDir() {
					cfg.logWriter.Printf("Ignoring excluded directory %s and its children", path)
					return fs.SkipDir
				}
				cfg.logWriter.Printf("Skipping excluded file %s", path)
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
					cfg.dependentModules[filepath.Clean(dir)] = false
				}
				numSkipped++
				return nil
			}
			// This is the mandatory format of unit test file names.
			if cfg.skipTestFiles && strings.HasSuffix(baseFile, "_test.go") {
				if cfg.logWriter.VerboseLevel(1) {
					cfg.logWriter.Printf("Skipping test file %s", path)
				}
				numSkipped++
				return nil
			}

			if cfg.skipProtoBufFiles && strings.HasSuffix(baseFile, ".pb.go") {
				if cfg.logWriter.VerboseLevel(1) {
					cfg.logWriter.Printf("Skipping generated file %s", path)
				}
				numSkipped++
				return nil
			}

			paths = append(paths, path)

			return nil
		})
	if err != nil {
		err = fmt.Errorf("error walking input directory %s: %v", cfg.InputDirectory, err)
	}

	return
}


func (cfg *Config) GetNotifierDirectory() string {
	return cfg.notifierDirectory
}

func (cfg *Config) GetCatalogBaseDir() string {
	return cfg.catalogBaseDir
}

func (cfg *Config) ShowDependentModules() {
	isText := ""
	cfg.logWriter.Printf("")
	cfg.logWriter.Printf("Module Usage Summary")
	for modName, used := range cfg.dependentModules {
		isText = "is"
		if !used {
			isText = "is not"
		}
		cfg.logWriter.Printf("%s %s used", modName, isText)
	}
	cfg.logWriter.Printf("")
}

func (cfg *Config) UpdateDependentModules(file_name string) {
	ok := false
	isUsed := false
	this_dir := filepath.Clean(filepath.Dir(file_name))
	for !ok {
		if cfg.logWriter.VerboseLevel(2) {
			cfg.logWriter.Printf("Checking if %q is a dependentModule", this_dir)
		}
		if this_dir == "." {
			break
		}
		isUsed, ok = cfg.dependentModules[this_dir]
		if ok {
			if !isUsed {
				cfg.dependentModules[this_dir] = true
			}
			return
		} else {
			old_dir := this_dir
			this_dir = filepath.Clean(filepath.Dir(this_dir))
			ok = (old_dir == this_dir)
		}
	}
	cfg.logWriter.Printf("%q does not belong to a scanned module", file_name)
}

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
