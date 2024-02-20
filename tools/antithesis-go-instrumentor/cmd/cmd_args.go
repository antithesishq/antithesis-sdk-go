package cmd

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/mod/modfile"
  "github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
)

// Capitalized struct items are accessed outside this file
type CommandArgs struct {
	ShowVersion         bool
	InvalidArgs         bool
  logWriter           *common.LogWriter
	excludeFile         string
	symPrefix           string
	wantsInstrumentor   bool
	catalogDir          string
	inputDir            string
	outputDir           string
	instrumentorVersion string
}

func ParseArgs(versionText string) *CommandArgs {
	versionPtr := flag.Bool("version", false, "the current version of this application")
	exclusionsPtr := flag.String("exclude", "", "the path to a file listing files and directories to exclude from instrumentation (optional)")
	prefixPtr := flag.String("prefix", "", "a string to prepend to the symbol table (optional)")
	logfilePtr := flag.String("logfile", "", "file path to log into (default=stderr)")
	verbosePtr := flag.Int("V", 0, "verbosity level (default to 0)")
	assertOnlyPtr := flag.Bool("assert_only", false, "generate assertion catalog ONLY - no coverage instrumentation (default to false)")
	catalogDirPtr := flag.String("catalog_dir", "", "file path where assertion catalog will be generated")
  instrVersionPtr := flag.String("instrumentor_version", "@latest", "version of the SDK instrumentation package to require")
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

	// Verify we have the expected number of positional arguments
	numArgsRequired := 1
	if cmdArgs.wantsInstrumentor {
		numArgsRequired++
	}

	if flag.NArg() < numArgsRequired {
		fmt.Fprintf(os.Stderr, strings.TrimSpace(versionText))
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

func (ca *CommandArgs) NewCommandFiles() (err error, cfx *CommandFiles) {
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
		err, moduleName = GetModuleName(customerInputDirectory)
		if err != nil {
			err = fmt.Errorf("Unable to obtain go module name from %q", customerInputDirectory)
		}
	}

	customerDirectory := ""
	symbolsDirectory := ""
	if ca.wantsInstrumentor {
		customerDirectory = filepath.Join(outputDirectory, "customer")
		symbolsDirectory = filepath.Join(outputDirectory, "symbols")
		if err == nil {
			err = CreateOutputDirectories(customerDirectory, symbolsDirectory)
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
		symbolsDirectory:    symbolsDirectory,
		catalogPath:         catalogPath,
		excludeFile:         ca.excludeFile,
		wantsInstrumentor:   ca.wantsInstrumentor,
		symtablePrefix:      symtablePrefix,
    instrumentorVersion: ca.instrumentorVersion,
    logWriter:           common.GetLogWriter(),
	}
	return
}

func GetModuleName(inputDir string) (err error, moduleName string) {
	var moduleData []byte
	moduleName = ""
	var f *modfile.File = nil
	moduleFilenamePath := filepath.Join(inputDir, "go.mod")
	if moduleData, err = os.ReadFile(moduleFilenamePath); err != nil {
		return
	}

	if f, err = modfile.ParseLax("go.mod", moduleData, nil); err == nil {
		moduleName = filepath.Base(f.Module.Mod.Path)
	}
	return
}

func CreateOutputDirectories(customerDirectory, symbolsDirectory string) (err error) {
	if err = os.Mkdir(customerDirectory, 0755); err != nil {
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
