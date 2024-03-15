package main

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/antithesishq/antithesis-sdk-go/internal"
	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/assertions"
	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/cmd"
	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
)

var logWriter *common.LogWriter

//go:embed version.txt
var versionText string

func main() {
	var err error

	versionString := strings.TrimSpace(versionText)
	if strings.Contains(versionText, "%s") {
		versionString = fmt.Sprintf(versionString, internal.SDK_Version)
	}

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
	if cmd_files, err = cmd_args.NewCommandFiles(); err != nil {
		logWriter.Printf(err.Error())
		os.Exit(1)
	}

	var source_files []string
	if source_files, err = cmd_files.GetSourceFiles(); err != nil {
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
	// Wrap-up processing and generate assertions catalog and notifier module
	//--------------------------------------------------------------------------------
	edge_count := cI.WrapUp()
	if edge_count > 0 {
		notifierDir := cmd_files.GetNotifierDirectory()
		cI.WriteNotifierSource(notifierDir, edge_count)
		cmd_files.CreateNotifierModule()
	}

	if aScanner.HasAssertionsDefined() {
		aScanner.WriteAssertionCatalog(cmd_args.VersionText)
	}
	cmd_files.WrapUp()

	//--------------------------------------------------------------------------------
	// Summarize results in logger
	//--------------------------------------------------------------------------------
	cI.SummarizeWork(len(source_files))
	aScanner.SummarizeWork()
}
