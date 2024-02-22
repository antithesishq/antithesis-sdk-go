package common

import (
	// "bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const HashBitsUsed = 48
const HashBytesUsed = HashBitsUsed / 8
const EncodedHashByteLength = HashBytesUsed * 2

// HashFileContent reads the binary content of
// every file in paths (assumed to be in lexical order)
// and returns the SHA-256 digest.
func HashFileContent(paths []string) string {
	hasher := sha256.New()
	for _, path := range paths {
		bytes, err := ioutil.ReadFile(path)
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
	commandLine := fmt.Sprintf("cp -v --no-clobber --recursive %s/* %s", from, to)
	cmd := exec.Command("bash", "-c", commandLine)
	logWriter.Printf("Executing %s", commandLine)
	allOutput, err := cmd.CombinedOutput()
  allText := strings.TrimSpace(string(allOutput))
  lines := strings.Split(allText, "\n")
  for _, line := range lines {
    logWriter.Printf("cp: %s", line)
  }
	if err != nil {
    logWriter.Printf("Ignoring cp exit code: %+v", err)
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

func canonicalizeDirectory(d string) string {
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
	input = canonicalizeDirectory(input) + "/"
	output = canonicalizeDirectory(output) + "/"
	if strings.HasPrefix(output, input) {
		err = fmt.Errorf("The input directory %s is a prefix of the output directory %s", input, output)
		return
	}
	if strings.HasPrefix(input, output) {
		err = fmt.Errorf("The output directory %s is a prefix of the input directory %s", output, input)
	}
	return
}

func createOutputDirectories(outputDirectory string) (string, string) {
	confirmEmptyOutputDirectory(outputDirectory)
	customer := filepath.Join(outputDirectory, "customer")
	symbols := filepath.Join(outputDirectory, "symbols")

	if e := os.Mkdir(customer, 0755); e != nil {
		logWriter.Fatal(e)
	}
	if e := os.Mkdir(symbols, 0755); e != nil {
		logWriter.Fatal(e)
	}

	return customer, symbols
}
