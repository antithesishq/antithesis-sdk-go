package main

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/antithesishq/antithesis-sdk-go/internal"
	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/args"
	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/config"
	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/scanners/assertions"
	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/scanners/coverage"
	covconfig "github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/scanners/coverage/config"
)

var logWriter *common.LogWriter

//go:embed version.txt
var versionText string

func main() {
	versionString := strings.TrimSpace(versionText)
	if strings.Contains(versionText, "%s") {
		versionString = fmt.Sprintf(versionString, internal.SDK_Version)
	}

	//--------------------------------------------------------------------------------
	// Parse and validate command arguments
	// Establish global logging
	//--------------------------------------------------------------------------------
	thisVersion := fmt.Sprintf("v%s", internal.SDK_Version)
	parsedArgs := args.ParseArgs(versionString, thisVersion)
	if parsedArgs.ShowVersion {
		fmt.Println(strings.TrimSpace(versionString))
		os.Exit(0)
	}
	if parsedArgs.InvalidArgs {
		os.Exit(1)
	}

	logWriter = common.GetLogWriter()
	logWriter.Printf("%s", strings.TrimSpace(versionString))
	parsedArgs.ShowArguments()

	//--------------------------------------------------------------------------------
	// Verify Directories and Files are all as expected
	// Prepare instrumentation output directories
	//--------------------------------------------------------------------------------
	cov, err := covconfig.NewCoverageConfig(parsedArgs)
	if err != nil {
		logWriter.Printf("%s", err.Error())
		os.Exit(1)
	}

	cc, err := config.NewCommonConfig(parsedArgs)
	if err != nil {
		logWriter.Printf("%s", err.Error())
		os.Exit(1)
	}

	var source_files []string
	if source_files, err = cov.GetSourceFiles(cc); err != nil {
		logWriter.Printf("%s", err.Error())
		os.Exit(1)
	}

	//--------------------------------------------------------------------------------
	// Setup coverage processor
	//--------------------------------------------------------------------------------
	cI := coverage.NewCoverageInstrumentor(cc, cov)
	source_dir := cc.GetSourceDir()
	target_dir := cc.GetTargetDir()

	//--------------------------------------------------------------------------------
	// Pass 1: Coverage instrumentation (file-by-file)
	//--------------------------------------------------------------------------------
	cov.ShowDependentModules(cc)
	for _, file_name := range source_files {
		if assertions.IsGeneratedFile(file_name) {
			logWriter.Printf("Skipping %s", file_name)
			continue
		}

		if instrumented_source := cI.InstrumentFile(file_name); instrumented_source != "" {
			cI.WriteInstrumentedOutput(cc, file_name, instrumented_source)
			cov.UpdateDependentModules(cc, file_name)
		}
	}
	cov.ShowDependentModules(cc)

	//--------------------------------------------------------------------------------
	// Wrap-up coverage instrumentation and generate notifier module
	//--------------------------------------------------------------------------------
	edge_count := cI.WrapUp()
	if edge_count > 0 {
		notifierDir := cov.GetNotifierDirectory()
		cI.WriteNotifierSource(notifierDir, edge_count)
		cov.CreateNotifierModule(cc)
	}

	//--------------------------------------------------------------------------------
	// Pass 2: Assertion catalog generation (go/packages-based, per-binary)
	//--------------------------------------------------------------------------------
	aScanner := assertions.NewAssertionScanner(logWriter.IsVerbose(), source_dir, target_dir)
	if err := aScanner.ScanAll(); err != nil {
		logWriter.Printf("Assertion scanning failed: %s", err.Error())
		logWriter.Printf("Assertion catalogs will not be generated")
	} else if aScanner.HasAssertionsDefined() {
		aScanner.WriteAssertionCatalogs(parsedArgs.VersionText)
	}

	cov.WrapUp(cc)

	//--------------------------------------------------------------------------------
	// Summarize results in logger
	//--------------------------------------------------------------------------------
	cI.SummarizeWork(len(source_files))
	aScanner.SummarizeWork()
}
