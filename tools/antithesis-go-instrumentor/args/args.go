package args

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
)

// Args holds the parsed command-line arguments.
type Args struct {
	logWriter         *common.LogWriter
	ExcludeFile       string
	SymPrefix         string
	InputDir          string
	OutputDir         string
	InstrumentorVersion string
	LocalSDKPath      string
	VersionText       string
	ShowVersion       bool
	InvalidArgs       bool
	WantsInstrumentor bool
	SkipTestFiles     bool
	SkipProtoBufFiles bool
}

func ParseArgs(versionText string, thisVersion string) *Args {
	versionPtr := flag.Bool("version", false, "the current version of this application")
	exclusionsPtr := flag.String("exclude", "", "the path to a file listing files and directories to exclude from instrumentation (optional)")
	prefixPtr := flag.String("prefix", "", "a string to prepend to the symbol table (optional)")
	logfilePtr := flag.String("logfile", "", "file path to log into (default=stderr)")
	verbosePtr := flag.Int("V", 0, "verbosity level (default to 0)")
	assertOnlyPtr := flag.Bool("assert_only", false, "generate assertion catalog ONLY - no coverage instrumentation (default to false)")
	catalogDirPtr := flag.String("catalog_dir", "", "(deprecated, ignored)")
	instrVersionPtr := flag.String("instrumentor_version", thisVersion, "version of the SDK instrumentation package to require")
	localSDKPathPtr := flag.String("local_sdk_path", "", "path to the local Antithesis SDK")
	skipTestFilesPtr := flag.Bool("skip_test_files", false, "Skip instrumentation and cataloging for '*_test.go' files (default to false)")
	skipProtoBufFilesPtr := flag.Bool("skip_protobuf_files", false, "Skip instrumentation and cataloging for '*.pb.go' files (default to false)")
	flag.Parse()

	args := Args{
		InvalidArgs: false,
		ShowVersion: *versionPtr,
	}

	if args.ShowVersion {
		return &args
	}

	args.logWriter = common.NewLogWriter(*logfilePtr, *verbosePtr)
	args.WantsInstrumentor = !*assertOnlyPtr
	args.SymPrefix = strings.TrimSpace(*prefixPtr)
	args.ExcludeFile = strings.TrimSpace(*exclusionsPtr)
	args.InstrumentorVersion = strings.TrimSpace(*instrVersionPtr)
	args.LocalSDKPath = strings.TrimSpace(*localSDKPathPtr)
	args.VersionText = versionText
	args.SkipTestFiles = *skipTestFilesPtr
	args.SkipProtoBufFiles = *skipProtoBufFilesPtr

	// Verify we have the expected number of positional arguments
	numArgsRequired := 1
	if args.WantsInstrumentor {
		numArgsRequired++
	}

	if *catalogDirPtr != "" {
		args.logWriter.Printf("Warning: -catalog_dir is deprecated and will be ignored")
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
		fmt.Fprintf(os.Stderr, "  antithesis_catalog.go\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "  - For assertions support, the catalog will be created in the go_project_dir\n")
		fmt.Fprintf(os.Stderr, "  - For full instrumentation, the catalog will be created under the target_dir\n")
		fmt.Fprintf(os.Stderr, "\n\n")
		flag.Usage()
		args.InvalidArgs = true
		return &args
	}

	if args.SymPrefix != "" {
		m, _ := regexp.MatchString(`^[a-z]+$`, *prefixPtr)
		if !m {
			fmt.Fprint(os.Stderr, "A prefix must consist of lower-case ASCII letters.\n")
			args.InvalidArgs = true
			return &args
		}
	}

	args.InputDir = flag.Arg(0)
	if args.WantsInstrumentor {
		args.OutputDir = flag.Arg(1)
	}

	if !isGoAvailable() {
		fmt.Fprint(os.Stderr, "Go toolchain not available\n")
		args.InvalidArgs = true
	}

	return &args
}

func (args *Args) ShowArguments() {
	args.logWriter.Printf("inputDir: %q", args.InputDir)
	if args.LocalSDKPath != "" {
		args.logWriter.Printf("localSDKPath: %q", args.LocalSDKPath)
	}
	if args.WantsInstrumentor {
		args.logWriter.Printf("outputDir: %q", args.OutputDir)
		if args.ExcludeFile != "" {
			args.logWriter.Printf("excludeFile: %q", args.ExcludeFile)
		}
		if args.SymPrefix != "" {
			args.logWriter.Printf("symPrefix: %q", args.SymPrefix)
		}
	}

	// Intentional: no need to show anything if not skipping
	if args.SkipTestFiles {
		args.logWriter.Printf("skipTestFiles: %t", args.SkipTestFiles)
	}
	if args.SkipProtoBufFiles {
		args.logWriter.Printf("skipProtoBufFiles: %t", args.SkipProtoBufFiles)
	}
}

func isGoAvailable() bool {
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
