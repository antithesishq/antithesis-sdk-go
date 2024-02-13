package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"flag"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/mod/modfile"
)

//go:embed version.txt
var versionString string

// ------------------------------------------------------------
// Replaces glog
//
// If the verbosity at the call site is less than or equal to
// level requested, the log will be enabled.  Higher callsite
// verbosity values are less likely to be output.
//
// if (2 <= verbosity) { log-is-enabled }
//
// Warning: \u261d
// Error: \u274c
// ------------------------------------------------------------
var logger *log.Logger
var verbosity int = 0

func verbose_level(v int) bool {
	return (v <= verbosity)
}

// FindSourceCode scans an input directory recursively for .go files,
// skipping any files or directories specified in exclusions.
func FindSourceCode(inputDirectory string, exclusions map[string]bool) []string {
	paths := []string{}
	logger.Printf("Scanning %s recursively for .go source", inputDirectory)
	// Files are read in lexical order, i.e. we can later deterministically
	// hash their content: https://pkg.go.dev/path/filepath#WalkDir
	err := filepath.WalkDir(inputDirectory,
		func(path string, info fs.DirEntry, err error) error {
			if err != nil {
				logger.Printf("\u274c Error %v in directory %s; skipping", err, path)
				return err
			}

			if b := filepath.Base(path); strings.HasPrefix(b, ".") {
				if verbose_level(2) {
					logger.Printf("Skipping %s", path)
				}
				if info.IsDir() {
					return fs.SkipDir
				}
				return nil
			}

			if exclusions[path] {
				if info.IsDir() {
					logger.Printf("Skipping excluded directory %s and its children", path)
					return fs.SkipDir
				}
				logger.Printf("Skipping excluded file %s", path)
				return nil
			}
			if info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			// This is the mandatory format of unit test file names.
			if strings.HasSuffix(path, "_test.go") {
				if verbose_level(2) {
					logger.Printf("Skipping test file %s", path)
				}
				return nil
			} else if strings.HasSuffix(path, ".pb.go") {
				if verbose_level(1) {
					logger.Printf("Skipping generated file %s", path)
				}
				return nil
			}

			paths = append(paths, path)

			return nil
		})

	if err != nil {
		logger.Fatalf("Error walking input directory %s: %v", inputDirectory, err)
	}

	return paths
}

// HashFileContent reads the binary content of
// every file in paths (assumed to be in lexical order)
// and returns the SHA-256 digest.
func HashFileContent(paths []string) string {
	hasher := sha256.New()
	for _, path := range paths {
		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			logger.Fatalf("Error reading file %s: %v", path, err)
		}
		hasher.Write(bytes)
	}

	return hex.EncodeToString(hasher.Sum(nil))[0:12]
}

func validateInputAsModule(path string) {
	moduleFile := filepath.Join(path, "go.mod")
	if _, err := os.ReadFile(moduleFile); err != nil {
		logger.Fatalf("There was no readable go.mod file at %s: %v", path, err)
	}
}

func validateAntithesisModule(path string) {
	antithesisModuleFile := filepath.Join(path, "go.mod")
	file, err := os.Open(antithesisModuleFile)
	if err != nil {
		logger.Fatalf("There was no readable go.mod file at %s: %v", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		logger.Fatalf("There was no readable go.mod file at %s: %v", path, err)
	}
	if scanner.Text() != "module antithesis.com/go/instrumentation" {
		logger.Fatalf("%s does not appear to be the go.mod for the Antithesis wrapper", antithesisModuleFile)
	}
}

func verifyGoOnPath() {
	logger.Printf("Confirming that go is on $PATH...")
	cmd := exec.Command("go", "version")
	_, err := cmd.Output()
	if err != nil {
		logger.Fatalf("%v", err)
	}
}

func copyRecursiveNoClobber(from, to string) {
	commandLine := fmt.Sprintf("cp --update=none --recursive %s/* %s", from, to)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("bash", "-c", commandLine)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	logger.Printf("%q", commandLine)
	err := cmd.Run()
	if err != nil {
		logger.Fatalf("%+v", err)
	}
}

func addDependencies(customerInputDirectory /* antithesisOutputDirectory, */, customerOutputDirectory string) {
	// old_commandLine := fmt.Sprintf("(cd %s; go mod edit -require=antithesis.com/go/instrumentation@v1.0.0 -replace antithesis.com/go/instrumentation=%s -print > %s/go.mod)",
	// customerInputDirectory,
	// antithesisOutputDirectory,
	// customerOutputDirectory)
	commandLine := fmt.Sprintf("(cd %s; go mod edit -require=github.com/antithesishq/antithesis-sdk-go/instrumentation@latest -print > %s/go.mod)",
		customerInputDirectory,
		customerOutputDirectory)

	cmd := exec.Command("bash", "-c", commandLine)
	logger.Printf("%s", commandLine)
	_, err := cmd.Output()
	if err != nil {
		// Errors here are pretty mysterious.
		logger.Fatalf("%v", err)
	}
}

func writeInstrumentedSource(source, path string) error {
	// Any errors here are fatal anyway, so I'm not checking.
	f, e := os.Create(path)
	if e != nil {
		logger.Printf("\u261d could not create %s", path)
		return e
	}
	defer f.Close()
	_, e = f.WriteString(source)
	if e != nil {
		logger.Printf("\u261d Could not write instrumented source to %s", path)
		return e
	}
	return nil
}

func main() {
	versionPtr := flag.Bool("version", false, "the current version of this application")
	exclusionsPtr := flag.String("exclude", "", "the path to a file listing files and directories to exclude from instrumentation (optional)")
	// antithesisPtr := flag.String("antithesis", "", "the directory containing the Antithesis instrumentation wrappers (required)")
	prefixPtr := flag.String("prefix", "", "a string to prepend to the symbol table (optional)")
	logfilePtr := flag.String("logfile", "", "file path to log into (default=stderr)")
	verbosePtr := flag.Int("V", 0, "verbosity level (default to 0)")
	assertOnlyPtr := flag.Bool("assert_only", false, "generate assertion catalog ONLY - no coverage instrumentation (default to false)")
	catalogDirPtr := flag.String("catalog_dir", "", "file path where assertion catalog will be generated")
	flag.Parse()

	if *versionPtr {
		fmt.Println(strings.TrimSpace(versionString))
		os.Exit(0)
	}

	wrx := os.Stderr
	if logfilePtr != nil {
		if fp, erx := os.Create(*logfilePtr); erx == nil {
			wrx = fp
		}
	}
	logger = log.New(wrx, "", log.LstdFlags|log.Lshortfile)

	if verbosePtr != nil {
		verbosity = *verbosePtr
	}

	want_instrumentor := !*assertOnlyPtr
	num_args_required := 1
	if want_instrumentor {
		num_args_required++
	}

	if flag.NArg() < num_args_required {
		flag.Usage()
		fmt.Fprint(os.Stderr, "\nThis program requires:\n")
		fmt.Fprintf(os.Stderr, "- An input directory of Golang source to be instrumented\n")
		if num_args_required > 1 {
			fmt.Fprintf(os.Stderr, "- An output directory for the instrumented results\n")
		}
		os.Exit(1)
	}

	// No longer need this
	// if *antithesisPtr == "" {
	// 	flag.Usage()
	// 	os.Exit(1)
	// }

	if *prefixPtr != "" {
		m, _ := regexp.MatchString(`^[a-z]+$`, *prefixPtr)
		if !m {
			fmt.Fprint(os.Stderr, "A prefix must consist of lower-case ASCII letters.")
			os.Exit(1)
		}
	}

	logger.Println(strings.TrimSpace(versionString))

	verifyGoOnPath()

	outputDirectory := ""
	customerInputDirectory := GetAbsoluteDirectory(flag.Arg(0))
	if want_instrumentor {
		outputDirectory = GetAbsoluteDirectory(flag.Arg(1))
		ValidateDirectories(customerInputDirectory, outputDirectory)
	}

	validateInputAsModule(customerInputDirectory)

	// no longer using this
	// validateAntithesisModule(*antithesisPtr)

	customerOutputDirectory := ""
	symbolsOutputDirectory := ""

	if want_instrumentor {
		// customerOutputDirectory, antithesisOutputDirectory, symbolsOutputDirectory := createOutputDirectories(outputDirectory)
		customerOutputDirectory, symbolsOutputDirectory = createOutputDirectories(outputDirectory)
	}

	exclusions := map[string]bool{}
	if *exclusionsPtr != "" {
		exclusions = ParseExclusionsFile(*exclusionsPtr, customerInputDirectory)
	}

	sourceFiles := FindSourceCode(customerInputDirectory, exclusions)

	hash := HashFileContent(sourceFiles)[0:12]
	// Each module has to have a generated name, and, per Go's rules,
	// be put in a directory with that name.
	// shimPkgBase := "instrumented_module_" + hash
	shimPkg := InstrumentationModuleName

	// No longer need this either
	// shimDirectory := filepath.Join(antithesisOutputDirectory, shimPkgBase)
	// shimPath := filepath.Join(shimDirectory, "instrumented_module.go")

	// No longer need the shim
	// if e := os.MkdirAll(shimDirectory, 0700); e != nil {
	// 	logger.Fatalf("Could not create subdirectory for Antithesis shim %s: %v", shimPath, e)
	// }

	if want_instrumentor {
		logger.Printf("Instrumenting %s to %s", customerInputDirectory, customerOutputDirectory)
	}

	symbolTableFileBaseName := "go-" + hash
	if *prefixPtr != "" {
		symbolTableFileBaseName = *prefixPtr + "-" + symbolTableFileBaseName
	}
	var instrumentor *Instrumentor = nil
	var symbolTableWriter *SymbolTable = nil

	usingSymbols := ""
	symbolTableFileName := symbolTableFileBaseName + ".sym.tsv"
	if want_instrumentor {
		usingSymbols = symbolTableFileName
		symbolsPath := filepath.Join(symbolsOutputDirectory, symbolTableFileName)
		symbolTableWriter = CreateSymbolTableFile(symbolsPath, symbolTableFileBaseName)
		instrumentor = CreateInstrumentor(customerInputDirectory, shimPkg, symbolTableWriter)
	}

	// Obtain the module name we are instrumenting
	var module_data []byte
	var e2 error = nil
	var f *modfile.File = nil
	module_filename_path := filepath.Join(customerInputDirectory, "go.mod")
	if module_data, e2 = os.ReadFile(module_filename_path); e2 != nil {
		logger.Fatalf("Could not access go.mod file: %q", module_filename_path)
	}

	if f, e2 = modfile.ParseLax("go.mod", module_data, nil); e2 != nil {
		panic(e2)
	}
	module_name := filepath.Base(f.Module.Mod.Path)

	// If not specified, the catalog will be written to:
	// (assert_only == true) ? the customerInputDirectory
	// want_instrumentor ? the customerOutputDirectory
	catalogDir := *catalogDirPtr
	if len(catalogDir) == 0 {
		catalogDir = customerInputDirectory
		if want_instrumentor {
			catalogDir = customerOutputDirectory
		}
	}

	full_catalog_path := filepath.Join(catalogDir, module_name)

	// Setup the assertion scanner (used to create the assertion catalog)
	aSI := NewScanningInfo(verbosity > 0, full_catalog_path, usingSymbols)

	filesInstrumented := 0
	filesCataloged := 0
	previousEdge := 0 // instrumentor.CurrentEdge
	instrumented := ""
	var e error = nil

	for _, path := range sourceFiles {
		base_name := filepath.Base(path)
		if path_was_generated := strings.HasSuffix(base_name, GENERATED_SUFFIX); path_was_generated {
			logger.Printf("Skipping %s", path)
			continue
		}
		if want_instrumentor {
			logger.Printf("Instrumenting %s", path)
			previousEdge = instrumentor.CurrentEdge
			instrumented, e = instrumentor.Instrument(path)

			if e != nil {
				logger.Printf("\u274c File %s produced error %s; simply copying source", path, e)
				continue
			}

			if instrumented == "" {
				// The instrumentor should have reported why it didn't instrument this file.
				continue
			}
		}

		// Scan for assertions
		logger.Printf("Cataloging %s", path)
		aSI.ScanFile(path)
		filesCataloged++

		if want_instrumentor {
			// Strip the prefix from the input file name. We could also use strings.Rel(),
			// but we've got absolute paths, so this will work.
			outputPath := filepath.Join(customerOutputDirectory, path[len(customerInputDirectory):])
			outputSubdirectory := filepath.Dir(outputPath)
			os.MkdirAll(outputSubdirectory, 0755)

			if verbose_level(1) {
				logger.Printf("Writing instrumented file %s with edges %d–%d", outputPath, previousEdge, instrumentor.CurrentEdge)
			}

			if e = writeInstrumentedSource(instrumented, outputPath); e == nil {
				filesInstrumented++
			}
		}
	}

	if want_instrumentor {
		if err := symbolTableWriter.Close(); err != nil {
			logger.Printf("\u274c Could not close symbol table %s: %s", symbolTableWriter.Path, err)
		}
		logger.Printf("Wrote symbol table %s", symbolTableWriter.Path)

		// TODO: Shim should be added into the file containing the Assertion catalog
		// Make sure this works when there are no assertions whatsoever
		// writeShimSource(instrumentor.CurrentEdge, shimPkgBase, symbolTableFileName, shimPath)
		// logger.Printf("Antithesis instrumentation shim written to %s", shimPath)

		// TODO: Dont need these wrapper files anymore
		// copyRecursiveNoClobber(*antithesisPtr, antithesisOutputDirectory)
		// logger.Printf("Antithesis instrumentation module %s copied to %s", *antithesisPtr, antithesisOutputDirectory)

		// TODO: Dependencies will just be the Antithesis SDK
		addDependencies(customerInputDirectory /* antithesisOutputDirectory,*/, customerOutputDirectory)
		logger.Printf("Antithesis dependencies added to %s/go.mod", customerOutputDirectory)

		copyRecursiveNoClobber(customerInputDirectory, customerOutputDirectory)
		logger.Printf("All other files copied unmodified from %s to %s", customerInputDirectory, customerOutputDirectory)

		num_files_read := len(sourceFiles)
		num_files_skipped := num_files_read - filesInstrumented
		logger.Printf("%d '.go' files read, %d files skipped, %d edges instrumented",
			num_files_read,
			num_files_skipped,
			instrumentor.CurrentEdge)
	}
	edge_count := 0
	if want_instrumentor {
		edge_count = instrumentor.CurrentEdge
	}
	aSI.WriteAssertionCatalog(edge_count)
	logger.Printf("%d '.go' files cataloged", filesCataloged)
}

func writeShimSource(currentEdge int, shimPkg string, symbolTable string, shimPath string) {
	f, err := os.Create(shimPath)
	if err != nil {
		logger.Fatalf("Could not open wrapper file %s: %v", shimPath, err)
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	CreateShimSource(shimPkg, symbolTable, currentEdge, w)
	w.Flush()
}

func copyFile(sourcePath string, destinationPath string) {
	inputBytes, e := ioutil.ReadFile(sourcePath)
	e = ioutil.WriteFile(destinationPath, inputBytes, 0644)
	if e != nil {
		logger.Printf("\u274c creating %s: %v", destinationPath, e)
	}
}
