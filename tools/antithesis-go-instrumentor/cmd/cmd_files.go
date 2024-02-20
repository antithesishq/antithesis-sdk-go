package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

  "github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
  "github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/instrumentor"
)

// Capitalized struct items are accessed outside this file
type CommandFiles struct {

	// The base directry of a go module to be instrumented/cataloged
	// Contains a go.mod file
	inputDirectory      string
	
	// The instrumentation (only) base oputput directory
	// Is required to exist, and to be empty prior to instrumentation
	//
	// After instrumentation, will contain the subdirectories for 
	// 'symbols' and 'customer'
	outputDirectory     string
	
	// created and written to during instrumentation.
	// Will contain a copy of the inputDirectory, where All
	// non-excluded '.go' files are instrumented
	customerDirectory   string
	
	// created and written during instrumentation.
	// Will contain the corresponding .tsv file expected
	// by the antithesis fuzzer
	symbolsDirectory    string
	
	// The directory that the generated assertion catalog
	// will be written to.  By default, this file will be 
	// written to the 'inputDirectory' when instrumentation
	// is not performed.  If instrumentation is performed,
	// the generated assertion catalog will be written to
	// the customerDirectory.  In both cases, the catalogPath 
	// is used to directly specify what directory the
	// generated assertion catalog should be written to.
	catalogPath         string
	
	// Indicates that instrumentation is requested (true)
	// If set to (false) then perform assertion catalog scanning
	// without instrumentation, which is common
	// when execution is outside of the Antithesis environment
	wantsInstrumentor   bool
	
	// A prefix used to distinguish symbol table filenames
	// that will be used by the antithesis fuzzer.
	symtablePrefix      string
	
	// The name of the symbol table file, which incorporates
	// the overall 'filesHash' and 'symbtablePrefix'
	symbolTableFilename string
	
	// Option file containing a list (one per line) of
	// any files or directories to be excluded from both
	// instumentation and assertion scanning.  Empty lines
	// and lines beginining with '#' are ignored.
	excludeFile         string
	
	// The set of exclusions that were obtained from
	// reading the 'excludeFile'. A map is used where
	// the value is always 'true', in lieu of a specific
	// 'set' abstraction not available for the version
	// of go used for this tool.
	exclusions          map[string]bool
	
	// All of the files (after exclusions) to be instrumented
	// and scanned for assertions that should appear in the
	// assertion catalog
	sourceFiles         []string
	
	// SHA256 Hash (48-bits worth) of all the files
	// in sourceFiles
	filesHash           string
	
	// Number of files skipped when creating the sourceFiles
	// list.
	filesSkipped        int

	// The version of SDK to use at runtime for CoverageInstrumentation
	instrumentorVersion string

  // Global logger
  logWriter           *common.LogWriter
}

func (cfx *CommandFiles) GetSourceFiles() (err error, sourceFiles []string) {
	sourceFiles = []string{}
	cfx.filesSkipped = 0
	if err = cfx.ParseExclusionsFile(); err != nil {
		return
	}

	numSkipped := 0
	if err, sourceFiles, numSkipped = cfx.FindSourceCode(); err != nil {
		return
	}
	cfx.filesSkipped = numSkipped

	cfx.filesHash = common.HashFileContent(sourceFiles)
	return
}

func (cfx *CommandFiles) NewCoverageInstrumentor() *instrumentor.CoverageInstrumentor {

	var file_instrumentor *instrumentor.Instrumentor
	var symTable *instrumentor.SymbolTable

	if cfx.wantsInstrumentor {
		cfx.logWriter.Printf("Writing instrumented source to %s", cfx.customerDirectory)
		symTable = cfx.CreateSymbolTableWriter(cfx.filesHash)
		file_instrumentor = instrumentor.CreateInstrumentor(cfx.inputDirectory, instrumentor.InstrumentationModuleName, symTable)
	}

	cI := instrumentor.CoverageInstrumentor{
		GoInstrumentor:    file_instrumentor,
		SymTable:          symTable,
		UsingSymbols:      cfx.UsingSymbols(),
		FullCatalogPath:   cfx.catalogPath,
		PreviousEdge:      0,
		FilesInstrumented: 0,
		FilesSkipped:      cfx.filesSkipped,
	}
	return &cI
}

func (cfx *CommandFiles) WrapUp() {
	if !cfx.wantsInstrumentor {
		return
	}
	// Dependencies will just be the Antithesis SDK
	common.AddDependencies(cfx.inputDirectory, cfx.customerDirectory, cfx.instrumentorVersion)
	cfx.logWriter.Printf("Antithesis dependencies added to %s/go.mod", cfx.customerDirectory)

	common.CopyRecursiveNoClobber(cfx.inputDirectory, cfx.customerDirectory)
	cfx.logWriter.Printf("All other files copied unmodified from %s to %s", cfx.inputDirectory, cfx.customerDirectory)
}

func (cfx *CommandFiles) WriteInstrumentedOutput(fileName string, instrumentedSource string, cI *instrumentor.CoverageInstrumentor) {
	// skip over the base inputDirectory from the inputfilename,
	// and create the output directories needed
	skipLength := len(cfx.inputDirectory)
	outputPath := filepath.Join(cfx.customerDirectory, fileName[skipLength:])
	outputSubdirectory := filepath.Dir(outputPath)
	os.MkdirAll(outputSubdirectory, 0755)

	if cfx.logWriter.VerboseLevel(1) {
		cfx.logWriter.Printf("Writing instrumented file %s with edges %d–%d", outputPath, cI.PreviousEdge, cI.GoInstrumentor.CurrentEdge)
	}

	if err := common.WriteTextFile(instrumentedSource, outputPath); err == nil {
		cI.FilesInstrumented++
	}
	return
}

func (cfx *CommandFiles) ParseExclusionsFile() (err error) {
	if cfx.excludeFile == "" {
		return
	}
	cfx.exclusions = map[string]bool{}
	var parsedExclusions map[string]bool

	err, parsedExclusions = ParseExclusionsFile(cfx.excludeFile, cfx.inputDirectory)
	if err == nil {
		cfx.exclusions = parsedExclusions
	}
	return
}

// FindSourceCode scans an input directory recursively for .go files,
// skipping any files or directories specified in exclusions.
func (cfx *CommandFiles) FindSourceCode() (err error, paths []string, numSkipped int) {
	paths = []string{}
	numSkipped = 0
	cfx.logWriter.Printf("Scanning %s recursively for .go source", cfx.inputDirectory)
	// Files are read in lexical order, i.e. we can later deterministically
	// hash their content: https://pkg.go.dev/path/filepath#WalkDir
	err = filepath.WalkDir(cfx.inputDirectory,
		func(path string, info fs.DirEntry, erx error) error {
			if erx != nil {
				cfx.logWriter.Printf("Error %v in directory %s; skipping", erx, path)
				return erx
			}

			if b := filepath.Base(path); strings.HasPrefix(b, ".") {
				if cfx.logWriter.VerboseLevel(2) {
					cfx.logWriter.Printf("Ignoring 'dot' directory: %s", path)
				}
				if info.IsDir() {
					return fs.SkipDir
				}
				return nil
			}

			if cfx.exclusions[path] {
				if info.IsDir() {
					cfx.logWriter.Printf("Ignoring excluded directory %s and its children", path)
					return fs.SkipDir
				}
				cfx.logWriter.Printf("Skipping excluded file %s", path)
				numSkipped++
				return nil
			}
			if info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				numSkipped++
				return nil
			}
			// This is the mandatory format of unit test file names.
			if strings.HasSuffix(path, "_test.go") {
				if cfx.logWriter.VerboseLevel(2) {
					cfx.logWriter.Printf("Skipping test file %s", path)
				}
				numSkipped++
				return nil
			} else if strings.HasSuffix(path, ".pb.go") {
				if cfx.logWriter.VerboseLevel(1) {
					cfx.logWriter.Printf("Skipping generated file %s", path)
				}
				numSkipped++
				return nil
			}

			paths = append(paths, path)

			return nil
		})

	if err != nil {
		err = fmt.Errorf("Error walking input directory %s: %v", cfx.inputDirectory, err)
	}
	return
}

func (cfx *CommandFiles) UsingSymbols() string {
	usingSymbols := ""
	if cfx.wantsInstrumentor {
		usingSymbols = cfx.symbolTableFilename
	}
	return usingSymbols
}

func (cfx *CommandFiles) CreateSymbolTableWriter(filesHash string) (symWriter *instrumentor.SymbolTable) {
  var err error
	cfx.symbolTableFilename = ""
	if cfx.wantsInstrumentor {
		symbolTableFileBasename := fmt.Sprintf("%sgo-%s", cfx.symtablePrefix, filesHash)
		cfx.symbolTableFilename = symbolTableFileBasename + ".sym.tsv"
		symbolsPath := filepath.Join(cfx.symbolsDirectory, cfx.symbolTableFilename)
		err, symWriter = instrumentor.CreateSymbolTableFile(symbolsPath, symbolTableFileBasename)
    if err != nil {
      cfx.logWriter.Fatalf("Could not write symbol table header: %s", err.Error())
    }
	}
	return
}
