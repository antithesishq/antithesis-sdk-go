package main

// Capitalized struct items are accessed outside this file
type CoverageInstrumentor struct {
	GoInstrumentor    *Instrumentor
	symTable          *SymbolTable
	UsingSymbols      string
	FullCatalogPath   string
	PreviousEdge      int
	FilesInstrumented int
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

func (cI *CoverageInstrumentor) SummarizeWork(num_files int) {
	if cI.GoInstrumentor == nil {
		return
	}
	num_files_skipped := num_files - cI.FilesInstrumented
	logger.Printf("%d '.go' file(s), %d file(s) skipped, %d edge(s) instrumented",
		num_files,
		num_files_skipped,
		cI.GoInstrumentor.CurrentEdge)
}
