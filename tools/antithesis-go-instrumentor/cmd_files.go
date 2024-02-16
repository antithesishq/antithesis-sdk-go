package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Capitalized struct items are accessed outside this file
type CommandFiles struct {
	outputDirectory     string
	inputDirectory      string
	customerDirectory   string
	symbolsDirectory    string
	catalogPath         string
	exclusions          map[string]bool
	excludeFile         string
	wantsInstrumentor   bool
	symtablePrefix      string
	symbolTableFilename string
	sourceFiles         []string
	filesHash           string
	filesSkipped        int
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

	cfx.filesHash = HashFileContent(sourceFiles)[0:12]
	return
}

func (cfx *CommandFiles) NewCoverageInstrumentor() *CoverageInstrumentor {

	var instrumentor *Instrumentor = nil
	var symTable *SymbolTable = nil

	if cfx.wantsInstrumentor {
		logger.Printf("Instrumenting %s to %s", cfx.inputDirectory, cfx.customerDirectory)
		symTable = cfx.CreateSymbolTableWriter(cfx.filesHash)
		instrumentor = CreateInstrumentor(cfx.inputDirectory, InstrumentationModuleName, symTable)
	}

	cI := CoverageInstrumentor{
		GoInstrumentor:    instrumentor,
		symTable:          symTable,
		UsingSymbols:      cfx.UsingSymbols(),
		FullCatalogPath:   cfx.catalogPath,
		PreviousEdge:      0,
		FilesInstrumented: 0,
		filesSkipped:      cfx.filesSkipped,
	}
	return &cI
}

func (cfx *CommandFiles) WrapUp() {
	if !cfx.wantsInstrumentor {
		return
	}
	// Dependencies will just be the Antithesis SDK
	addDependencies(cfx.inputDirectory, cfx.customerDirectory)
	logger.Printf("Antithesis dependencies added to %s/go.mod", cfx.customerDirectory)

	copyRecursiveNoClobber(cfx.inputDirectory, cfx.customerDirectory)
	logger.Printf("All other files copied unmodified from %s to %s", cfx.inputDirectory, cfx.customerDirectory)
}

func (cfx *CommandFiles) WriteInstrumentedOutput(fileName string, instrumentedSource string, cI *CoverageInstrumentor) {
	skipLength := len(cfx.inputDirectory)
	outputPath := filepath.Join(cfx.customerDirectory, fileName[skipLength:])
	outputSubdirectory := filepath.Dir(outputPath)
	os.MkdirAll(outputSubdirectory, 0755)

	if VerboseLevel(1) {
		logger.Printf("Writing instrumented file %s with edges %d–%d", outputPath, cI.PreviousEdge, cI.GoInstrumentor.CurrentEdge)
	}

	if err := writeTextFile(instrumentedSource, outputPath); err == nil {
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
	logger.Printf("Scanning %s recursively for .go source", cfx.inputDirectory)
	// Files are read in lexical order, i.e. we can later deterministically
	// hash their content: https://pkg.go.dev/path/filepath#WalkDir
	err = filepath.WalkDir(cfx.inputDirectory,
		func(path string, info fs.DirEntry, erx error) error {
			if erx != nil {
				logger.Printf("Error %v in directory %s; skipping", erx, path)
				return erx
			}

			if b := filepath.Base(path); strings.HasPrefix(b, ".") {
				if VerboseLevel(2) {
					logger.Printf("Ignoring 'dot' directory: %s", path)
				}
				if info.IsDir() {
					return fs.SkipDir
				}
				return nil
			}

			if cfx.exclusions[path] {
				if info.IsDir() {
					logger.Printf("Ignoring excluded directory %s and its children", path)
					return fs.SkipDir
				}
				logger.Printf("Skipping excluded file %s", path)
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
				if VerboseLevel(2) {
					logger.Printf("Skipping test file %s", path)
				}
				numSkipped++
				return nil
			} else if strings.HasSuffix(path, ".pb.go") {
				if VerboseLevel(1) {
					logger.Printf("Skipping generated file %s", path)
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

func (cfx *CommandFiles) CreateSymbolTableWriter(filesHash string) (symWriter *SymbolTable) {
	symWriter = nil
	cfx.symbolTableFilename = ""
	if cfx.wantsInstrumentor {
		symbolTableFileBasename := fmt.Sprintf("%sgo-%s", cfx.symtablePrefix, filesHash)
		cfx.symbolTableFilename = symbolTableFileBasename + ".sym.tsv"
		symbolsPath := filepath.Join(cfx.symbolsDirectory, cfx.symbolTableFilename)
		symWriter = CreateSymbolTableFile(symbolsPath, symbolTableFileBasename)
	}
	return
}
