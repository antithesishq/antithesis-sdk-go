package main 

import (
	_ "embed"
  "flag"
  "fmt"
  "log"
  "os"
  "os/exec"
  "path/filepath"
  "regexp"
  "strings"

	"golang.org/x/mod/modfile"
)

type CommandArgs struct {
  ShowVersion bool 
  InvalidArgs bool
  ExcludeFile string
  Prefix string 
  Logger *log.Logger
  Verbosity int 
  WantsInstrumentor bool
  CatalogDir string 
  InputDir string
  OutputDir string
}

//go:embed version.txt
var versionString string

func parse_args() *CommandArgs {
	versionPtr := flag.Bool("version", false, "the current version of this application")
	exclusionsPtr := flag.String("exclude", "", "the path to a file listing files and directories to exclude from instrumentation (optional)")
	prefixPtr := flag.String("prefix", "", "a string to prepend to the symbol table (optional)")
	logfilePtr := flag.String("logfile", "", "file path to log into (default=stderr)")
	verbosePtr := flag.Int("V", 0, "verbosity level (default to 0)")
	assertOnlyPtr := flag.Bool("assert_only", false, "generate assertion catalog ONLY - no coverage instrumentation (default to false)")
	catalogDirPtr := flag.String("catalog_dir", "", "file path where assertion catalog will be generated")
	flag.Parse()

  cmd_args := CommandArgs{
    InvalidArgs: false,
    ShowVersion: *versionPtr,
  }

	if cmd_args.ShowVersion {
    return &cmd_args
	}

	wrx := os.Stderr
  logfile_path := strings.TrimSpace(*logfilePtr)
	if logfile_path != "" {
		if fp, erx := os.Create(logfile_path); erx == nil {
			wrx = fp
		}
	}
  cmd_args.Logger = log.New(wrx, "", log.LstdFlags|log.Lshortfile)
  cmd_args.Verbosity = *verbosePtr
	cmd_args.WantsInstrumentor = !*assertOnlyPtr
  cmd_args.Prefix = strings.TrimSpace(*prefixPtr)
  cmd_args.CatalogDir = strings.TrimSpace(*catalogDirPtr)
  cmd_args.ExcludeFile = strings.TrimSpace(*exclusionsPtr)

  // Verify we have the non-flag arguments we expect
	num_args_required := 1
	if cmd_args.WantsInstrumentor {
		num_args_required++
	}

	if flag.NArg() < num_args_required {
		flag.Usage()
		fmt.Fprint(os.Stderr, "\nThis program requires:\n")
		fmt.Fprintf(os.Stderr, "- An input directory of Golang source to be instrumented\n")
		if num_args_required > 1 {
			fmt.Fprintf(os.Stderr, "- An output directory for the instrumented results\n")
		}
    cmd_args.InvalidArgs = true
    return &cmd_args
	}

	if cmd_args.Prefix != "" {
		m, _ := regexp.MatchString(`^[a-z]+$`, *prefixPtr)
		if !m {
			fmt.Fprint(os.Stderr, "A prefix must consist of lower-case ASCII letters.")
    	cmd_args.InvalidArgs = true
    	return &cmd_args
		}
	}

  cmd_args.InputDir = flag.Arg(0)
  if cmd_args.WantsInstrumentor {
    cmd_args.OutputDir = flag.Arg(1)
  }

  return &cmd_args
}

func (ca *CommandArgs) NewCommandFiles() (err error, cfx *CommandFiles) {
  outputDirectory := ""
  customerInputDirectory := GetAbsoluteDirectory(ca.InputDir)
  if ca.WantsInstrumentor {
    outputDirectory = GetAbsoluteDirectory(ca.OutputDir)
    err = ValidateDirectories(customerInputDirectory, outputDirectory)
  }

  symtable_prefix := ""
  if ca.Prefix != "" {
    symtable_prefix = ca.Prefix + "-"
  }

  module_name := ""
  if err == nil {
    err, module_name = GetModuleName(customerInputDirectory)
    if err != nil {
      err = fmt.Errorf("Unable to obtain go module name from %q", customerInputDirectory)
    }
  }

  customerDirectory := filepath.Join(outputDirectory, "customer")
  symbolsDirectory := filepath.Join(outputDirectory, "symbols")
  if err == nil {
    CreateOutputDirectories(customerDirectory, symbolsDirectory)
  }

	catalogDir := ca.CatalogDir
	if catalogDir == "" {
		catalogDir = customerInputDirectory
		if ca.WantsInstrumentor {
			catalogDir = customerDirectory
		}
	}
	catalog_path := filepath.Join(catalogDir, module_name)

  cfx = &CommandFiles{
    OutputDirectory: outputDirectory,
    InputDirectory: customerInputDirectory,
    CustomerDirectory: customerDirectory,
    SymbolsDirectory: symbolsDirectory,
    CatalogPath: catalog_path,
    exclusions: make(map[string]bool),
    exclude_file: ca.ExcludeFile,
    wants_instrumentor: ca.WantsInstrumentor,
    module_name: module_name,
    symtable_prefix: symtable_prefix,
    symbolTableFilename: "",
  }
  return
}


func GetModuleName(input_dir string) (err error, module_name string) {
	var module_data []byte
  module_name = ""
	var f *modfile.File = nil
	module_filename_path := filepath.Join(input_dir, "go.mod")
	if module_data, err = os.ReadFile(module_filename_path); err != nil {
    return 
	}

	if f, err = modfile.ParseLax("go.mod", module_data, nil); err == nil {
    module_name = filepath.Base(f.Module.Mod.Path)
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
  parts := strings.Split(strings.TrimSpace(string(output)), " ")
  if len(parts) < 4 {
    return false
  }
  return (parts[0] == "go") && (parts[1] == "version") && strings.HasPrefix(parts[2], "go")
}
