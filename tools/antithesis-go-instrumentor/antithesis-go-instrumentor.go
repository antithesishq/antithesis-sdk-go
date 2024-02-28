package main

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/assertions"
	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/cmd"
	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
)

var logWriter *common.LogWriter

//go:embed version.txt
var versionString string

func main() {
	var err error

	//--------------------------------------------------------------------------------
	// Parse and validate command arguments
	// Establish global logging
	//--------------------------------------------------------------------------------
	cmd_args := cmd.ParseArgs(versionString)
	if cmd_args.ShowVersion {
		fmt.Println(strings.TrimSpace(versionString))
		os.Exit(0)
	}
	if cmd_args.InvalidArgs {
		os.Exit(1)
	}

	logWriter = common.GetLogWriter()
	logWriter.Printf(strings.TrimSpace(versionString))

	//--------------------------------------------------------------------------------
	// Verify Directories and Files are all as expected
	// Prepare instrumentation output directories
	//--------------------------------------------------------------------------------
	var cmd_files *cmd.CommandFiles
	if err, cmd_files = cmd_args.NewCommandFiles(); err != nil {
		logWriter.Printf(err.Error())
		os.Exit(1)
	}

	var source_files []string
	if err, source_files = cmd_files.GetSourceFiles(); err != nil {
		logWriter.Printf(err.Error())
		os.Exit(1)
	}

	//--------------------------------------------------------------------------------
	// Setup coverage and assertion processors
	//--------------------------------------------------------------------------------
	cI := cmd_files.NewCoverageInstrumentor()
	aScanner := assertions.NewAssertionScanner(logWriter.IsVerbose(), cI.FullCatalogPath, cI.UsingSymbols)

	//--------------------------------------------------------------------------------
	// Process all files (ignore previously generated assertion catalogs)
	//--------------------------------------------------------------------------------
	for _, file_name := range source_files {
		if assertions.IsGeneratedFile(file_name) {
			logWriter.Printf("Skipping %s", file_name)
			continue
		}

		if instrumented_source := cI.InstrumentFile(file_name); instrumented_source != "" {
			cmd_files.WriteInstrumentedOutput(file_name, instrumented_source, cI)
		}

		aScanner.ScanFile(file_name)
	}

	//--------------------------------------------------------------------------------
	// Wrap-up processing, summarize results in logger
	//--------------------------------------------------------------------------------
	cmd_files.WrapUp(aScanner.HasAssertionsDefined())
	edge_count := cI.WrapUp()
	cI.SummarizeWork(len(source_files))

	aScanner.WriteAssertionCatalog(edge_count)
	aScanner.SummarizeWork()
}
