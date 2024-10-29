package common

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	HashBitsUsed          = 48
	HashBytesUsed         = HashBitsUsed / 8
	EncodedHashByteLength = HashBytesUsed * 2
)

// HashFileContent reads the binary content of
// every file in paths (assumed to be in lexical order)
// and returns the SHA-256 digest.
func HashFileContent(paths []string) string {
	hasher := sha256.New()
	for _, path := range paths {
		bytes, err := os.ReadFile(path)
		if err != nil {
			logWriter.Fatalf("Error reading file %s: %v", path, err)
		}
		hasher.Write(bytes)
	}

	return hex.EncodeToString(hasher.Sum(nil))[0:EncodedHashByteLength]
}

func WriteTextFile(text, file_name string) (err error) {
	var f *os.File
	if f, err = os.Create(file_name); err != nil {
		logWriter.Printf("Error: could not create %s", file_name)
		return
	}
	defer f.Close()
	if _, err = f.WriteString(text); err != nil {
		logWriter.Printf("Error: Could not write text to %s", file_name)
	}
	return
}

func CopyRecursiveNoClobber(from, to string) {
	commandLine := fmt.Sprintf("cp -n -R %s/* %s", from, to)
	cmd := exec.Command("bash", "-c", commandLine)
	logWriter.Printf("")
	logWriter.Printf("Copying all other files (non-instrumented files)")
	logWriter.Printf("Executing %s", commandLine)
	allOutput, err := cmd.CombinedOutput()
	allText := strings.TrimSpace(string(allOutput))
	lines := strings.Split(allText, "\n")
	for _, line := range lines {
		if len(line) > 0 {
			logWriter.Printf("cp: %s", line)
		}
	}
	if err != nil {
		logWriter.Printf("cp completed with %+v", err)
	}
}

func AddDependencies(customerInputDirectory, customerOutputDirectory, instrumentorVersion, notifierModule, localNotifier string) {
	destGoModFile := fmt.Sprintf("%s/go.mod", customerOutputDirectory)

	cmd1 := fmt.Sprintf("cd %s", customerInputDirectory)
	cmd2 := fmt.Sprintf("go mod edit -require=%s@v0.0.0 -replace=%s=%s -print > %s",
		notifierModule, notifierModule, localNotifier, destGoModFile)
	commandLine := fmt.Sprintf("(%s; %s)", cmd1, cmd2)

	cmd := exec.Command("bash", "-c", commandLine)
	logWriter.Printf("Adding Antithesis module as a dependency to %s", customerOutputDirectory)
	logWriter.Printf("Executing %s", commandLine)
	allOutput, err := cmd.CombinedOutput()
	allText := strings.TrimSpace(string(allOutput))
	if len(allText) > 0 {
		lines := strings.Split(allText, "\n")
		for _, line := range lines {
			logWriter.Printf("go mod edit: %s", line)
		}
	}
	if err != nil {
		// Errors here are pretty mysterious.
		logWriter.Fatalf("%v", err)
	}
}

func FetchDependencies(customerOutputDirectory string) {
	commandLine := fmt.Sprintf("(cd %s; go mod tidy)", customerOutputDirectory)

	cmd := exec.Command("bash", "-c", commandLine)
	logWriter.Printf("")
	logWriter.Printf("Fetching Dependencies (go mod tidy)")
	logWriter.Printf("Executing %s", commandLine)
	allOutput, err := cmd.CombinedOutput()
	allText := strings.TrimSpace(string(allOutput))
	if len(allText) > 0 {
		lines := strings.Split(allText, "\n")
		for _, line := range lines {
			logWriter.Printf("go mod tidy: %s", line)
		}
	}
	if err != nil {
		// Errors here are pretty mysterious.
		logWriter.Fatalf("%v", err)
	}
}

func NotifierDependencies(notifierOutputDirectory, notifierModuleName, instrumentorVersion, localSDKPath string) {
	dependencyRef := fmt.Sprintf("go get %s@%s",
		ANTITHESIS_SDK_MODULE,
		instrumentorVersion)

	if localSDKPath != "" {
		dependencyRef = fmt.Sprintf("go mod edit -require=%s@v0.0.0 -replace=%s=%s",
			ANTITHESIS_SDK_MODULE, ANTITHESIS_SDK_MODULE, localSDKPath)
	}

	commandLine := fmt.Sprintf("(cd %s; go mod init %s; %s; go mod tidy)",
		notifierOutputDirectory,
		notifierModuleName,
		dependencyRef)

	cmd := exec.Command("bash", "-c", commandLine)
	logWriter.Printf("")
	logWriter.Printf("Creating Notifier Module")
	logWriter.Printf("Executing %s", commandLine)
	allOutput, err := cmd.CombinedOutput()
	allText := strings.TrimSpace(string(allOutput))
	if len(allText) > 0 {
		lines := strings.Split(allText, "\n")
		for _, line := range lines {
			logWriter.Printf("go mod (notifier): %s", line)
		}
	}
	if err != nil {
		// Errors here are pretty mysterious.
		logWriter.Fatalf("%v", err)
	}
}

// GetAbsoluteDirectory converts a path, whether a symlink or
// a relative path, into an absolute path.
func GetAbsoluteDirectory(path string) string {
	if absolute, e := filepath.Abs(path); e != nil {
		logWriter.Fatalf("Could not evaluate %s as an absolute path: %v", path, e)
	} else {
		if s, err := os.Stat(absolute); err != nil {
			logWriter.Fatalf("%v", err)
		} else {
			if !s.IsDir() {
				logWriter.Fatalf("%s is not a directory", absolute)
			}
			return absolute
		}
	}
	// This code will never be executed.
	return ""
}

func CanonicalizeDirectory(d string) string {
	target, e := filepath.EvalSymlinks(d)
	if e != nil {
		logWriter.Fatalf("filepath.EvalSymlinks(%s) failed: %v", d, e)
	}

	a, e := filepath.Abs(target)
	if e != nil {
		logWriter.Fatalf("filepath.Abs(%s) failed: %v", target, e)
	}
	return a
}

func confirmEmptyOutputDirectory(output string) {
	d, e := os.Open(output)
	if e != nil {
		logWriter.Fatalf("Could not open %s: %v", output, e)
	}
	defer d.Close()
	// See the documentation on File.Readdirnames().
	if names, _ := d.Readdirnames(1); len(names) > 0 {
		logWriter.Fatalf("Output directory %s must be empty.", output)
	}
}

// ValidateDirectories checks that neither directory is a child of the other,
// and of course that they're not the same.
func ValidateDirectories(input, output string) (err error) {
	// Go does not have a type for filepaths, and will not do this for me: https://golang.org/src/path/filepath/path_unix.go?s=717:754#L16
	// The UNIX kernel absolutely forbids slashes in filenames. So, quick and dirty:
	input = CanonicalizeDirectory(input) + "/"
	output = CanonicalizeDirectory(output) + "/"
	if strings.HasPrefix(output, input) {
		err = fmt.Errorf("input directory %s is a prefix of the output directory %s", input, output)
		return
	}
	if strings.HasPrefix(input, output) {
		err = fmt.Errorf("output directory %s is a prefix of the input directory %s", output, input)
	}
	return
}

// PathFromBaseDirectory gets the path of someDir relative to baseDir
//
// Example:
// PathFromBaseDirectory("/home/ricky/etcd", "/home/ricky/etcd/server/test")
//
//	==> "server/test"
func PathFromBaseDirectory(baseDir, someDir string) string {
	baseNorm := CanonicalizeDirectory(baseDir)
	someNorm := CanonicalizeDirectory(someDir)
	if baseNorm == someNorm {
		return ""
	}
	someOffset := someNorm
	pattern := filepath.Join(baseNorm, "*")
	if didMatch, _ := filepath.Match(pattern, someNorm); didMatch {
		lx := len(baseNorm)
		idx := lx + 1
		if idx < len(someNorm) {
			someOffset = someNorm[idx:]
		}
	}
	return someOffset
}
