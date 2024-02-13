package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	// "github.com/golang/glog"
)

// ParseExclusionsFile reads the exclusions file, skipping lines beginning with
// #. Golang does not have a set class, so, rather than waste space copy-pastaing
// code from the interwebs, we'll just return a map.
func ParseExclusionsFile(path string, inputDirectory string) map[string]bool {
	exclusions := map[string]bool{}

	exclusionsFile, err := os.Open(path)
	if err != nil {
		// glog.Fatalf("Could not open exclusions %s: %v", path, err)
		logger.Fatalf("Could not open exclusions %s: %v", path, err)
	}
	defer exclusionsFile.Close()
	// glog.Infof("Reading exclusions from %s; relative paths will be resolved to %s", path, inputDirectory)
	logger.Printf("Reading exclusions from %s; relative paths will be resolved to %s", path, inputDirectory)
	scanner := bufio.NewScanner(exclusionsFile)
	for scanner.Scan() {
		entry := scanner.Text()
		if strings.HasPrefix(entry, "#") || strings.TrimSpace(entry) == "" {
			continue
		}

		exclusion := entry
		if !filepath.IsAbs(entry) {
			exclusion = filepath.Join(inputDirectory, entry)
		}

		exclusion, e := filepath.Abs(exclusion)
		if e != nil {
			// glog.Fatalf("Exclusion %s could not be resolved to an absolute path: %v", entry, e)
			logger.Fatalf("Exclusion %s could not be resolved to an absolute path: %v", entry, e)
		}

		if _, e := os.Stat(exclusion); e == nil {
			exclusions[exclusion] = true
			// glog.Infof("Exclusion %s added as %s", entry, exclusion)
			logger.Printf("Exclusion %s added as %s", entry, exclusion)
		} else {
			// glog.Fatalf("File %s in exclusions does not exist or is inaccessible", entry)
			logger.Fatalf("File %s in exclusions does not exist or is inaccessible", entry)
		}
	}

	if err := scanner.Err(); err != nil {
		// glog.Fatalf("Error scanning file %s: %v", path, err)
		logger.Fatalf("Error scanning file %s: %v", path, err)
	}

	return exclusions
}
