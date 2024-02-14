package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ------------------------------------------------------------
// Replaces glog
//
// If the verbosity at the call site is less than or equal to
// level requested, the log will be enabled.  Higher callsite
// verbosity values are less likely to be output.
//
// if (2 <= verbosity) { log-is-enabled }
// ------------------------------------------------------------
var logger *log.Logger
var verbosity int = 0

func verbose_level(v int) bool {
	return (v <= verbosity)
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

func addDependencies(customerInputDirectory, customerOutputDirectory string) {
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
		logger.Printf("Error: could not create %s", path)
		return e
	}
	defer f.Close()
	_, e = f.WriteString(source)
	if e != nil {
		logger.Printf("Error: Could not write instrumented source to %s", path)
		return e
	}
	return nil
}

func main() {
	cmd_args := parse_args()
	if cmd_args.ShowVersion {
		fmt.Println(strings.TrimSpace(versionString))
		os.Exit(0)
	}

	if cmd_args.InvalidArgs {
		os.Exit(1)
	}

	var err error

	// Setup logging globals
	// TODO Could also pass these settings inside a parameter (or inside a receiver)
	logger = cmd_args.Logger
	verbosity = cmd_args.Verbosity

	logger.Println(strings.TrimSpace(versionString))

	if !IsGoAvailable() {
		logger.Printf("Go toolchain not available")
		os.Exit(1)
	}

	var cmd_files *CommandFiles

	if err, cmd_files = cmd_args.NewCommandFiles(); err != nil {
		logger.Printf(err.Error())
		os.Exit(1)
	}

	if err = cmd_files.ParseExclusionsFile(); err != nil {
		logger.Printf(err.Error())
		os.Exit(1)
	}

	sourceFiles := []string{}
	if err, sourceFiles = cmd_files.FindSourceCode(); err != nil {
		logger.Printf(err.Error())
		os.Exit(1)
	}

	files_hash := HashFileContent(sourceFiles)[0:12]
	var instrumentor *Instrumentor = nil
	var symbolTableWriter *SymbolTable = nil

	if cmd_args.WantsInstrumentor {
		logger.Printf("Instrumenting %s to %s", cmd_files.InputDirectory, cmd_files.CustomerDirectory)
		symbolTableWriter = cmd_files.CreateSymbolTableWriter(files_hash)
		instrumentor = CreateInstrumentor(cmd_files.InputDirectory, InstrumentationModuleName, symbolTableWriter)
	}
	usingSymbols := cmd_files.UsingSymbols()
	full_catalog_path := cmd_files.CatalogPath

	// Setup the assertion scanner (used to create the assertion catalog)
	aSI := NewScanningInfo(verbosity > 0, full_catalog_path, usingSymbols)

	filesInstrumented := 0
	filesCataloged := 0
	previousEdge := 0
	instrumented := ""
	var e error = nil

	for _, path := range sourceFiles {
		base_name := filepath.Base(path)
		if path_was_generated := strings.HasSuffix(base_name, GENERATED_SUFFIX); path_was_generated {
			logger.Printf("Skipping %s", path)
			continue
		}
		if cmd_files.wants_instrumentor {
			logger.Printf("Instrumenting %s", path)
			previousEdge = instrumentor.CurrentEdge
			instrumented, e = instrumentor.Instrument(path)

			if e != nil {
				logger.Printf("Error: File %s produced error %s; simply copying source", path, e)
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

		if cmd_files.wants_instrumentor {
			// Strip the prefix from the input file name. We could also use strings.Rel(),
			// but we've got absolute paths, so this will work.

			outputPath := filepath.Join(cmd_files.CustomerDirectory, path[len(cmd_files.InputDirectory):])
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

	if cmd_files.wants_instrumentor {
		if err = symbolTableWriter.Close(); err != nil {
			logger.Printf("Error Could not close symbol table %s: %s", symbolTableWriter.Path, err)
		}
		logger.Printf("Wrote symbol table %s", symbolTableWriter.Path)

		// Dependencies will just be the Antithesis SDK
		addDependencies(cmd_files.InputDirectory, cmd_files.CustomerDirectory)
		logger.Printf("Antithesis dependencies added to %s/go.mod", cmd_files.CustomerDirectory)

		copyRecursiveNoClobber(cmd_files.InputDirectory, cmd_files.CustomerDirectory)
		logger.Printf("All other files copied unmodified from %s to %s", cmd_files.InputDirectory, cmd_files.CustomerDirectory)

		num_files_read := len(sourceFiles)
		num_files_skipped := num_files_read - filesInstrumented
		logger.Printf("%d '.go' files read, %d files skipped, %d edges instrumented",
			num_files_read,
			num_files_skipped,
			instrumentor.CurrentEdge)
	}
	edge_count := 0
	if cmd_files.wants_instrumentor {
		edge_count = instrumentor.CurrentEdge
	}
	aSI.WriteAssertionCatalog(edge_count)
	logger.Printf("%d '.go' files cataloged", filesCataloged)
}
