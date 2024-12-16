package cmd

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
	"golang.org/x/mod/modfile"
)

// Capitalized struct items are accessed outside this file
type CommandArgs struct {
	logWriter           *common.LogWriter
	excludeFile         string
	symPrefix           string
	catalogDir          string
	inputDir            string
	outputDir           string
	instrumentorVersion string
	localSDKPath        string
	VersionText         string
	ShowVersion         bool
	InvalidArgs         bool
	wantsInstrumentor   bool
	skipTestFiles       bool
	skipProtoBufFiles   bool
}

func ParseArgs(versionText string, thisVersion string) *CommandArgs {
	versionPtr := flag.Bool("version", false, "the current version of this application")
	exclusionsPtr := flag.String("exclude", "", "the path to a file listing files and directories to exclude from instrumentation (optional)")
	prefixPtr := flag.String("prefix", "", "a string to prepend to the symbol table (optional)")
	logfilePtr := flag.String("logfile", "", "file path to log into (default=stderr)")
	verbosePtr := flag.Int("V", 0, "verbosity level (default to 0)")
	assertOnlyPtr := flag.Bool("assert_only", false, "generate assertion catalog ONLY - no coverage instrumentation (default to false)")
	catalogDirPtr := flag.String("catalog_dir", "", "file path where assertion catalog will be generated")
	instrVersionPtr := flag.String("instrumentor_version", thisVersion, "version of the SDK instrumentation package to require")
	localSDKPathPtr := flag.String("local_sdk_path", "", "path to the local Antithesis SDK")
	skipTestFilesPtr := flag.Bool("skip_test_files", false, "Skip instrumentation and cataloging for '*_test.go' files (default to false)")
	skipProtoBufFilesPtr := flag.Bool("skip_protobuf_files", false, "Skip instrumentation and cataloging for '*.pb.go' files (default to false)")
	flag.Parse()

	cmdArgs := CommandArgs{
		InvalidArgs: false,
		ShowVersion: *versionPtr,
	}

	if cmdArgs.ShowVersion {
		return &cmdArgs
	}

	cmdArgs.logWriter = common.NewLogWriter(*logfilePtr, *verbosePtr)
	cmdArgs.wantsInstrumentor = !*assertOnlyPtr
	cmdArgs.symPrefix = strings.TrimSpace(*prefixPtr)
	cmdArgs.catalogDir = strings.TrimSpace(*catalogDirPtr)
	cmdArgs.excludeFile = strings.TrimSpace(*exclusionsPtr)
	cmdArgs.instrumentorVersion = strings.TrimSpace(*instrVersionPtr)
	cmdArgs.localSDKPath = strings.TrimSpace(*localSDKPathPtr)
	cmdArgs.VersionText = versionText
	cmdArgs.skipTestFiles = *skipTestFilesPtr
	cmdArgs.skipProtoBufFiles = *skipProtoBufFilesPtr

	// Verify we have the expected number of positional arguments
	numArgsRequired := 1
	if cmdArgs.wantsInstrumentor {
		numArgsRequired++
	}

	if flag.NArg() < numArgsRequired {
		fmt.Fprintf(os.Stderr, "%s", strings.TrimSpace(versionText))
		fmt.Fprintf(os.Stderr, "\n\n")
		fmt.Fprintf(os.Stderr, "For assertions support:\n")
		fmt.Fprintf(os.Stderr, "  $ antithesis-go-instrumentor -assert_only [options] go_project_dir\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "  - The go_project_dir should contain a valid go.mod file\n")
		fmt.Fprintf(os.Stderr, "\n\n")
		fmt.Fprintf(os.Stderr, "For full instrumentations (including assertions support):\n")
		fmt.Fprintf(os.Stderr, "  $ antithesis-go-instrumentor [options] go_project_dir target_dir\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "  - The go_project_dir should contain a valid go.mod file\n")
		fmt.Fprintf(os.Stderr, "  - The target_dir should be an existing, but empty directory\n")
		fmt.Fprintf(os.Stderr, "\n\n")
		fmt.Fprintf(os.Stderr, "The Assertions catalog will be registered in a generated file:\n")
		fmt.Fprintf(os.Stderr, "  <module-name>_antithesis_catalog.go\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "  - For assertions support, the catalog will be created in the go_project_dir\n")
		fmt.Fprintf(os.Stderr, "  - Override this directory using '-catalog_dir path_to-directory'\n")
		fmt.Fprintf(os.Stderr, "  - For full instrumentation, the catalog will be created under the target_dir\n")
		fmt.Fprintf(os.Stderr, "\n\n")
		flag.Usage()
		cmdArgs.InvalidArgs = true
		return &cmdArgs
	}

	if cmdArgs.symPrefix != "" {
		m, _ := regexp.MatchString(`^[a-z]+$`, *prefixPtr)
		if !m {
			fmt.Fprint(os.Stderr, "A prefix must consist of lower-case ASCII letters.\n")
			cmdArgs.InvalidArgs = true
			return &cmdArgs
		}
	}

	cmdArgs.inputDir = flag.Arg(0)
	if cmdArgs.wantsInstrumentor {
		cmdArgs.outputDir = flag.Arg(1)
	}

	if !IsGoAvailable() {
		fmt.Fprint(os.Stderr, "Go toolchain not available\n")
		cmdArgs.InvalidArgs = true
	}

	return &cmdArgs
}

func (ca *CommandArgs) ShowArguments() {
	ca.logWriter.Printf("inputDir: %q", ca.inputDir)
	if ca.localSDKPath != "" {
		ca.logWriter.Printf("localSDKPath: %q", ca.localSDKPath)
	}
	if ca.catalogDir != "" {
		ca.logWriter.Printf("catalogDir: %q", ca.catalogDir)
	}
	if ca.wantsInstrumentor {
		ca.logWriter.Printf("outputDir: %q", ca.outputDir)
		if ca.excludeFile != "" {
			ca.logWriter.Printf("excludeFile: %q", ca.excludeFile)
		}
		if ca.symPrefix != "" {
			ca.logWriter.Printf("symPrefix: %q", ca.symPrefix)
		}
	}

	// Intentional: no need to show anything if not skipping
	if ca.skipTestFiles {
		ca.logWriter.Printf("skipTestFiles: %t", ca.skipTestFiles)
	}
	if ca.skipProtoBufFiles {
		ca.logWriter.Printf("skipProtoBufFiles: %t", ca.skipProtoBufFiles)
	}
}

func (ca *CommandArgs) NewCommandFiles() (cfx *CommandFiles, err error) {
	outputDirectory := ""
	customerInputDirectory := common.GetAbsoluteDirectory(ca.inputDir)
	if ca.wantsInstrumentor {
		outputDirectory = common.GetAbsoluteDirectory(ca.outputDir)
		err = common.ValidateDirectories(customerInputDirectory, outputDirectory)
	}

	symtablePrefix := ""
	if ca.symPrefix != "" {
		symtablePrefix = ca.symPrefix + "-"
	}

	moduleName := ""
	if err == nil {
		if moduleName, err = GetModuleName(customerInputDirectory); err != nil {
			err = fmt.Errorf("unable to obtain go module name from %q", customerInputDirectory)
		}
	}

	customerDirectory := ""
	notifierDirectory := ""
	symbolsDirectory := ""
	if ca.wantsInstrumentor {
		customerDirectory = filepath.Join(outputDirectory, common.INSTRUMENTED_SOURCE_FOLDER)
		notifierDirectory = filepath.Join(outputDirectory, common.NOTIFIER_FOLDER)
		symbolsDirectory = filepath.Join(outputDirectory, common.SYMBOLS_FOLDER)
		if err == nil {
			err = CreateOutputDirectories(customerDirectory, notifierDirectory, symbolsDirectory)
		}
	}

	if err != nil {
		return
	}

	catalogDir := ca.catalogDir
	if catalogDir == "" {
		catalogDir = customerInputDirectory
		if ca.wantsInstrumentor {
			catalogDir = customerDirectory
		}
	}

	// It is possible that module names have "/" in their name
	// It is less likely they have "\" in their name
	// In either case, these characters are replaced with "_V_"
	// to compose the catalogPath. This catalogPath is used as the
	// main portion of a filepath which will contain the assertion
	// catalog. See details in function 'expectOutputFile' found
	// in 'catalog_output.go'
	tempName := strings.ReplaceAll(moduleName, "/", "_V_")
	flattenedModuleName := strings.ReplaceAll(tempName, "\\", "_V_")
	catalogPath := filepath.Join(catalogDir, flattenedModuleName)

	cfx = &CommandFiles{
		outputDirectory:     outputDirectory,
		inputDirectory:      customerInputDirectory,
		customerDirectory:   customerDirectory,
		notifierDirectory:   notifierDirectory,
		symbolsDirectory:    symbolsDirectory,
		catalogPath:         catalogPath,
		excludeFile:         ca.excludeFile,
		wantsInstrumentor:   ca.wantsInstrumentor,
		symtablePrefix:      symtablePrefix,
		instrumentorVersion: ca.instrumentorVersion,
		localSDKPath:        ca.localSDKPath,
		logWriter:           common.GetLogWriter(),
		skipTestFiles:       ca.skipTestFiles,
		skipProtoBufFiles:   ca.skipProtoBufFiles,
	}
	return
}

func GetModuleName(inputDir string) (moduleName string, err error) {
	var moduleData []byte
	moduleName = ""
	var f *modfile.File = nil
	moduleFilenamePath := filepath.Join(inputDir, "go.mod")
	if moduleData, err = os.ReadFile(moduleFilenamePath); err != nil {
		return
	}

	if f, err = modfile.ParseLax("go.mod", moduleData, nil); err == nil {
		moduleName = f.Module.Mod.Path
	}
	return
}

func CreateOutputDirectories(customerDirectory, notifierDirectory, symbolsDirectory string) (err error) {
	if err = os.Mkdir(customerDirectory, 0755); err != nil {
		return
	}
	if err = os.Mkdir(notifierDirectory, 0755); err != nil {
		return
	}
	err = os.Mkdir(symbolsDirectory, 0755)
	return
}

func IsGoAvailable() bool {
	cmd := exec.Command("go", "version")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// go version is expected to output 1 line containing 4 space-delimited items
	// Typical output expected is:
	//
	//   go version go1.21.5 linux/amd64
	//
	// verify we get this 'shape' output
	parts := strings.Split(strings.TrimSpace(string(output)), " ")
	if len(parts) < 4 {
		return false
	}
	return (parts[0] == "go") && (parts[1] == "version") && strings.HasPrefix(parts[2], "go")
}
