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
	thisVersion := fmt.Sprintf("v%s", internal.SDK_Version)
	cmd_args := cmd.ParseArgs(versionString, thisVersion)
	if cmd_args.ShowVersion {
		fmt.Println(strings.TrimSpace(versionString))
		os.Exit(0)
	}
	if cmd_args.InvalidArgs {
		os.Exit(1)
	}

	logWriter = common.GetLogWriter()
	logWriter.Printf("%s", strings.TrimSpace(versionString))
	cmd_args.ShowArguments()

	//--------------------------------------------------------------------------------
	// Verify Directories and Files are all as expected
	// Prepare instrumentation output directories
	//--------------------------------------------------------------------------------
	var cmd_files *cmd.CommandFiles
	if cmd_files, err = cmd_args.NewCommandFiles(); err != nil {
		logWriter.Printf("%s", err.Error())
		os.Exit(1)
	}

	var source_files []string
	if source_files, err = cmd_files.GetSourceFiles(); err != nil {
		logWriter.Printf("%s", err.Error())
		os.Exit(1)
	}

	//--------------------------------------------------------------------------------
	// Setup coverage processor
	//--------------------------------------------------------------------------------
	cI := cmd_files.NewCoverageInstrumentor()
	source_dir := cmd_files.GetSourceDir()
	target_dir := cmd_files.GetTargetDir()

	//--------------------------------------------------------------------------------
	// Pass 1: Coverage instrumentation (file-by-file)
	//--------------------------------------------------------------------------------
	cmd_files.ShowDependentModules()
	for _, file_name := range source_files {
		if assertions.IsGeneratedFile(file_name) {
			logWriter.Printf("Skipping %s", file_name)
			continue
		}

		if instrumented_source := cI.InstrumentFile(file_name); instrumented_source != "" {
			cmd_files.WriteInstrumentedOutput(file_name, instrumented_source, cI)
			cmd_files.UpdateDependentModules(file_name)
		}
	}
	cmd_files.ShowDependentModules()

	//--------------------------------------------------------------------------------
	// Wrap-up coverage instrumentation and generate notifier module
	//--------------------------------------------------------------------------------
	edge_count := cI.WrapUp()
	if edge_count > 0 {
		notifierDir := cmd_files.GetNotifierDirectory()
		cI.WriteNotifierSource(notifierDir, edge_count)
		cmd_files.CreateNotifierModule()
	}

	//--------------------------------------------------------------------------------
	// Pass 2: Assertion catalog generation (go/packages-based, per-binary)
	//--------------------------------------------------------------------------------
	aScanner := assertions.NewAssertionScanner(logWriter.IsVerbose(), source_dir, target_dir)
	if err := aScanner.ScanAll(); err != nil {
		logWriter.Printf("Assertion scanning failed: %s", err.Error())
		logWriter.Printf("Assertion catalogs will not be generated")
	} else if aScanner.HasAssertionsDefined() {
		aScanner.WriteAssertionCatalogs(cmd_args.VersionText)
	}

	cmd_files.WrapUp()

	//--------------------------------------------------------------------------------
	// Summarize results in logger
	//--------------------------------------------------------------------------------
	cI.SummarizeWork(len(source_files))
	aScanner.SummarizeWork()
}
