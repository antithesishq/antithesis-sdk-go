package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GetAbsoluteDirectory converts a path, whether a symlink or
// a relative path, into an absolute path.
func GetAbsoluteDirectory(path string) string {
	if absolute, e := filepath.Abs(path); e != nil {
		logger.Fatalf("Could not evaluate %s as an absolute path: %v", path, e)
	} else {
		if s, err := os.Stat(absolute); err != nil {
			logger.Fatalf("%v", err)
		} else {
			if !s.IsDir() {
				logger.Fatalf("%s is not a directory", absolute)
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
		logger.Fatalf("filepath.EvalSymlinks(%s) failed: %v", d, e)
	}

	a, e := filepath.Abs(target)
	if e != nil {
		logger.Fatalf("filepath.Abs(%s) failed: %v", target, e)
	}
	return a
}

func confirmEmptyOutputDirectory(output string) {
	d, e := os.Open(output)
	if e != nil {
		logger.Fatalf("Could not open %s: %v", output, e)
	}
	defer d.Close()
	// See the documentation on File.Readdirnames().
	if names, _ := d.Readdirnames(1); len(names) > 0 {
		logger.Fatalf("Output directory %s must be empty.", output)
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
	}
	if err == nil {
		if strings.HasPrefix(input, output) {
			err = fmt.Errorf("The output directory %s is a prefix of the input directory %s", output, input)
		}
	}
	return
}

func createOutputDirectories(outputDirectory string) (string, string) {
	confirmEmptyOutputDirectory(outputDirectory)
	customer := filepath.Join(outputDirectory, "customer")
	symbols := filepath.Join(outputDirectory, "symbols")

	if e := os.Mkdir(customer, 0755); e != nil {
		logger.Fatal(e)
	}
	if e := os.Mkdir(symbols, 0755); e != nil {
		logger.Fatal(e)
	}

	return customer, symbols
}
