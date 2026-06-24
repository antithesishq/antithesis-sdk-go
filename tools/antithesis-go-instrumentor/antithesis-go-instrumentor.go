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

	common.NewLogWriter(parsedArgs.LogFile, parsedArgs.VerbosityLevel)
	common.Logger.Printf(common.Normal, "%s", strings.TrimSpace(versionString))
	parsedArgs.ShowArguments()

	//--------------------------------------------------------------------------------
	// Verify Directories and Files are all as expected
	// Prepare instrumentation output directories
	//--------------------------------------------------------------------------------
	cc, err := config.NewCommonConfig(parsedArgs)
	if err != nil {
		common.Logger.Printf(common.Normal, "%s", err.Error())
		os.Exit(1)
	}

	source_dir := cc.GetSourceDir()
	target_dir := source_dir

	var cI *coverage.CoverageInstrumentor
	var numSourceFiles int

	//--------------------------------------------------------------------------------
	// Coverage instrumentation (only when not in assert-only mode)
	//--------------------------------------------------------------------------------
	if parsedArgs.WantsInstrumentor {
		cov, err := covconfig.NewCoverageConfig(parsedArgs)
		if err != nil {
			common.Logger.Printf(common.Normal, "%s", err.Error())
			os.Exit(1)
		}

		var source_files []string
		if source_files, err = cov.GetSourceFiles(cc); err != nil {
			common.Logger.Printf(common.Normal, "%s", err.Error())
			os.Exit(1)
		}
		numSourceFiles = len(source_files)

		cI = coverage.NewCoverageInstrumentor(cc, cov)
		target_dir = cc.CustomerDirectory

		// Pass 1: Coverage instrumentation (file-by-file)
		cov.ShowDependentModules()
		for _, file_name := range source_files {
			if assertions.IsGeneratedFile(file_name) {
				common.Logger.Printf(common.Normal, "Skipping %s", file_name)
				continue
			}

			if instrumented_source := cI.InstrumentFile(file_name); instrumented_source != "" {
				cI.WriteInstrumentedOutput(cc, file_name, instrumented_source)
				cov.UpdateDependentModules(file_name)
			}
		}
		cov.ShowDependentModules()

		// Wrap-up coverage instrumentation and generate notifier module
		edge_count := cI.WrapUp()
		if edge_count > 0 {
			notifierDir := cov.GetNotifierDirectory()
			cI.WriteNotifierSource(notifierDir, edge_count)
			cov.CreateNotifierModule(cc)
		}

		cov.WrapUp(cc)
	}

	//--------------------------------------------------------------------------------
	// Assertion catalog generation (go/packages-based, per-binary)
	//--------------------------------------------------------------------------------
	aScanner := assertions.NewAssertionScanner(source_dir, target_dir)
	if err := aScanner.ScanAll(); err != nil {
		common.Logger.Printf(common.Normal, "Assertion scanning failed: %s", err.Error())
		common.Logger.Printf(common.Normal, "Assertion catalogs will not be generated")
	} else if aScanner.HasAssertionsDefined() {
		aScanner.WriteAssertionCatalogs(parsedArgs.VersionText)
	}

	//--------------------------------------------------------------------------------
	// Summarize results in logger
	//--------------------------------------------------------------------------------
	if cI != nil {
		cI.SummarizeWork(numSourceFiles)
	}
	aScanner.SummarizeWork()
}
