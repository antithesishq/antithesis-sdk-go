package main

// Capitalized struct items are accessed outside this file
type CoverageInstrumentor struct {
	GoInstrumentor    *Instrumentor
	symTable          *SymbolTable
	UsingSymbols      string
	FullCatalogPath   string
	PreviousEdge      int
	FilesInstrumented int
	filesSkipped      int
}

func (cI *CoverageInstrumentor) InstrumentFile(file_name string) string {
	if cI.GoInstrumentor == nil {
		return ""
	}
	var err error
	instrumented := ""
	logger.Printf("Instrumenting %s", file_name)
	cI.PreviousEdge = cI.GoInstrumentor.CurrentEdge
	instrumented, err = cI.GoInstrumentor.Instrument(file_name)

	if err != nil {
		logger.Printf("Error: File %s produced error %s; simply copying source", file_name, err)
		return ""
	}

	return instrumented
}

func (cI *CoverageInstrumentor) WrapUp() (edge_count int) {
	var err error
	edge_count = 0

	if cI.GoInstrumentor != nil {
		if err = cI.symTable.Close(); err != nil {
			logger.Printf("Error Could not close symbol table %s: %s", cI.symTable.Path, err)
		}
		logger.Printf("Wrote symbol table %s", cI.symTable.Path)
		edge_count = cI.GoInstrumentor.CurrentEdge
	}
	return
}

func pluralize(val int, singularText string) string {
	if val == 1 {
		return singularText
	}
	return singularText + "s"
}

func (cI *CoverageInstrumentor) SummarizeWork(numFiles int) {
	if cI.GoInstrumentor == nil {
		return
	}

	numFilesSkipped := (numFiles - cI.FilesInstrumented) + cI.filesSkipped
	numEdges := cI.GoInstrumentor.CurrentEdge
	logger.Printf("%d '.go' %s instrumented, %d %s skipped, %d %s identified",
		numFiles, pluralize(numFiles, "file"),
		numFilesSkipped, pluralize(numFilesSkipped, "file"),
		numEdges, pluralize(numEdges, "edge"))
}
