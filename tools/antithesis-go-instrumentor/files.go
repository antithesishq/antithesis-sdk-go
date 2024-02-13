package main

import (
	"os"
	"path/filepath"
	"strings"
	// "github.com/golang/glog"
)

// GetAbsoluteDirectory converts a path, whether a symlink or
// a relative path, into an absolute path.
func GetAbsoluteDirectory(path string) string {
	if absolute, e := filepath.Abs(path); e != nil {
		// glog.Fatalf("Could not evaluate %s as an absolute path: %v", path, e)
		logger.Fatalf("Could not evaluate %s as an absolute path: %v", path, e)
	} else {
		if s, err := os.Stat(absolute); err != nil {
			// glog.Fatalf("%v", err)
			logger.Fatalf("%v", err)
		} else {
			if !s.IsDir() {
				// glog.Fatalf("%s is not a directory", absolute)
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
		// glog.Fatalf("filepath.EvalSymlinks(%s) failed: %v", d, e)
		logger.Fatalf("filepath.EvalSymlinks(%s) failed: %v", d, e)
	}

	a, e := filepath.Abs(target)
	if e != nil {
		// glog.Fatalf("filepath.Abs(%s) failed: %v", target, e)
		logger.Fatalf("filepath.Abs(%s) failed: %v", target, e)
	}
	return a
}

func confirmEmptyOutputDirectory(output string) {
	d, e := os.Open(output)
	if e != nil {
		// glog.Fatalf("Could not open %s: %v", output, e)
		logger.Fatalf("Could not open %s: %v", output, e)
	}
	defer d.Close()
	// See the documentation on File.Readdirnames().
	if names, _ := d.Readdirnames(1); len(names) > 0 {
		// glog.Fatalf("Output directory %s must be empty.", output)
		logger.Fatalf("Output directory %s must be empty.", output)
	}
}

// ValidateDirectories checks that neither directory is a child of the other,
// and of course that they're not the same.
func ValidateDirectories(input, output string) {
	// Go does not have a type for filepaths, and will not do this for me: https://golang.org/src/path/filepath/path_unix.go?s=717:754#L16
	// The UNIX kernel absolutely forbids slashes in filenames. So, quick and dirty:
	input = canonicalizeDirectory(input) + "/"
	output = canonicalizeDirectory(output) + "/"
	if strings.HasPrefix(output, input) {
		// glog.Fatalf("The input directory %s is a prefix of the output directory %s", input, output)
		logger.Fatalf("The input directory %s is a prefix of the output directory %s", input, output)
	}
	if strings.HasPrefix(input, output) {
		// glog.Fatalf("The output directory %s is a prefix of the input directory %s", output, input)
		logger.Fatalf("The output directory %s is a prefix of the input directory %s", output, input)
	}
}

func createOutputDirectories(outputDirectory string) (string, string) {
	confirmEmptyOutputDirectory(outputDirectory)
	customer := filepath.Join(outputDirectory, "customer")
	symbols := filepath.Join(outputDirectory, "symbols")

	// no longer used
	// antithesis := filepath.Join(outputDirectory, "antithesis")
	// if e := os.Mkdir(antithesis, 0755); e != nil {
	// 	logger.Fatal(e)
	// }

	if e := os.Mkdir(customer, 0755); e != nil {
		logger.Fatal(e)
	}
	if e := os.Mkdir(symbols, 0755); e != nil {
		logger.Fatal(e)
	}

	// return customer, antithesis, symbols
	return customer, symbols
}
