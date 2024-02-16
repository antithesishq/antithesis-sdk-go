package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	var err error

	//--------------------------------------------------------------------------------
	// Parse and validate command arguments
	// Establish global logging
	//--------------------------------------------------------------------------------
	cmd_args := ParseArgs()
	if cmd_args.ShowVersion {
		fmt.Println(strings.TrimSpace(versionString))
		os.Exit(0)
	}
	if cmd_args.InvalidArgs {
		os.Exit(1)
	}

	//--------------------------------------------------------------------------------
	// Verify Directories and Files are all as expected
	// Prepare instrumentation output directories
	//--------------------------------------------------------------------------------
	var cmd_files *CommandFiles
	if err, cmd_files = cmd_args.NewCommandFiles(); err != nil {
		logger.Printf(err.Error())
		os.Exit(1)
	}

	source_files := []string{}
	if err, source_files = cmd_files.GetSourceFiles(); err != nil {
		logger.Printf(err.Error())
		os.Exit(1)
	}

	//--------------------------------------------------------------------------------
	// Setup coverage and assertion processors
	//--------------------------------------------------------------------------------
	cI := cmd_files.NewCoverageInstrumentor()
	aSI := NewScanningInfo(verbosity > 0, cI.FullCatalogPath, cI.UsingSymbols)

	//--------------------------------------------------------------------------------
	// Process all files (ignore previously generated assertion catalogs)
	//--------------------------------------------------------------------------------
	for _, file_name := range source_files {
		if IsGeneratedFile(file_name) {
			logger.Printf("Skipping %s", file_name)
			continue
		}

		if instrumented_source := cI.InstrumentFile(file_name); instrumented_source != "" {
			cmd_files.WriteInstrumentedOutput(file_name, instrumented_source, cI)
		}

		aSI.ScanFile(file_name)
	}

	//--------------------------------------------------------------------------------
	// Wrap-up processing, summarize results in logger
	//--------------------------------------------------------------------------------
	cmd_files.WrapUp()
	edge_count := cI.WrapUp()
	cI.SummarizeWork(len(source_files))

	aSI.WriteAssertionCatalog(edge_count)
	aSI.SummarizeWork()
}
