package coverage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
	commonconfig "github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/config"
	covconfig "github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/scanners/coverage/config"
	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/scanners/coverage/instrumentor"
	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/scanners/coverage/symboltable"
)

// Capitalized struct items are accessed outside this file
type CoverageInstrumentor struct {
	GoInstrumentor    *instrumentor.Instrumentor
	SymTable          *symboltable.SymbolTable
	UsingSymbols    string
	NotifierPackage string
	PreviousEdge      int
	FilesInstrumented int
	FilesSkipped      int
}

type NotifierInfo struct {
	InstrumentationPackageName string
	SymbolTableName            string
	NotifierPackage            string
	EdgeCount                  int
}

func (cI *CoverageInstrumentor) WriteNotifierSource(notifierDir string, edge_count int) {
	notifierInfo := NotifierInfo{
		InstrumentationPackageName: common.InstrumentationPackageName(),
		SymbolTableName:            cI.UsingSymbols,
		EdgeCount:                  edge_count,
		NotifierPackage:            cI.NotifierPackage,
	}

	GenerateNotifierSource(notifierDir, &notifierInfo)
}

func (cI *CoverageInstrumentor) InstrumentFile(file_name string) string {
	var err error
	instrumented := ""
	common.Logger.Printf(common.Normal, "Instrumenting %s", file_name)
	cI.PreviousEdge = cI.GoInstrumentor.CurrentEdge
	if instrumented, err = cI.GoInstrumentor.Instrument(file_name); err != nil {
		common.Logger.Printf(common.Normal, "Error: File %s produced error %s; simply copying source", file_name, err)
		return ""
	}

	return instrumented
}

func (cI *CoverageInstrumentor) WrapUp() (edge_count int) {
	if err := cI.SymTable.Close(); err != nil {
		common.Logger.Printf(common.Normal, "Error Could not close symbol table %s: %s", cI.SymTable.Path, err)
	}
	common.Logger.Printf(common.Normal, "Symbol table: %s", cI.SymTable.Path)
	edge_count = cI.GoInstrumentor.CurrentEdge
	return
}

func (cI *CoverageInstrumentor) SummarizeWork(numFiles int) {
	numFilesSkipped := (numFiles - cI.FilesInstrumented) + cI.FilesSkipped
	numEdges := cI.GoInstrumentor.CurrentEdge
	common.Logger.Printf(common.Normal, "%d '.go' %s instrumented, %d %s skipped, %d %s identified",
		numFiles, common.Pluralize(numFiles, "file"),
		numFilesSkipped, common.Pluralize(numFilesSkipped, "file"),
		numEdges, common.Pluralize(numEdges, "edge"))
}

func NewCoverageInstrumentor(cc *commonconfig.CommonConfig, cov *covconfig.CoverageConfig) *CoverageInstrumentor {
	notifierModuleName := common.FullNotifierName(cc.FilesHash)

	common.Logger.Printf(common.Normal, "Writing instrumented source to %s", cc.CustomerDirectory)

	symbolTableFileBasename := fmt.Sprintf("%s%s-%s", cov.SymtablePrefix, common.SYMBOLS_FILE_HASH_PREFIX, cc.FilesHash)
	symbolTableFilename := symbolTableFileBasename + common.SYMBOLS_FILE_SUFFIX
	symbolsPath := filepath.Join(cov.SymbolsDirectory, symbolTableFilename)
	symTable, err := symboltable.CreateSymbolTableFile(symbolsPath, symbolTableFileBasename)
	if err != nil {
		common.Logger.Fatalf("Could not write symbol table header: %s", err.Error())
	}

	goInstrumentor := instrumentor.CreateInstrumentor(cc.InputDirectory, notifierModuleName, symTable)

	cI := CoverageInstrumentor{
		GoInstrumentor:    goInstrumentor,
		SymTable:          symTable,
		UsingSymbols:      symbolTableFilename,
		PreviousEdge:      0,
		FilesInstrumented: 0,
		FilesSkipped:      cc.FilesSkipped,
		NotifierPackage:   common.NotifierPackage(cc.FilesHash),
	}
	return &cI
}

func (cI *CoverageInstrumentor) WriteInstrumentedOutput(cc *commonconfig.CommonConfig, fileName string, instrumentedSource string) {
	// skip over the base inputDirectory from the inputfilename,
	// and create the output directories needed
	skipLength := len(cc.InputDirectory)
	outputPath := filepath.Join(cc.CustomerDirectory, fileName[skipLength:])
	outputSubdirectory := filepath.Dir(outputPath)
	os.MkdirAll(outputSubdirectory, 0755)

	common.Logger.Printf(common.Info, "Writing instrumented file %s with edges %d–%d", outputPath, cI.PreviousEdge, cI.GoInstrumentor.CurrentEdge)

	if err := common.WriteTextFile(instrumentedSource, outputPath); err == nil {
		cI.FilesInstrumented++
	}
}
