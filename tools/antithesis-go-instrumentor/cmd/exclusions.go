package cmd

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
)

// ParseExclusionsFile reads the exclusions file, skipping lines beginning with
// #. Golang does not have a set class, so, rather than waste space copy-pastaing
// code from the interwebs, we'll just return a map.
func ParseExclusionsFile(path string, inputDirectory string) (exclusions map[string]bool, err error) {
	exclusions = map[string]bool{}
	logWriter := common.GetLogWriter()
	var exclusionsFile *os.File
	if exclusionsFile, err = os.Open(path); err != nil {
		logWriter.Fatalf("Could not open exclusions %s: %v", path, err)
		return
	}
	defer exclusionsFile.Close()
	logWriter.Printf("Reading exclusions from %s; relative paths will be resolved to %s", path, inputDirectory)
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

		if exclusion, err = filepath.Abs(exclusion); err != nil {
			logWriter.Fatalf("Exclusion %s could not be resolved to an absolute path: %v", entry, err)
			return
		}

		if _, err = os.Stat(exclusion); err == nil {
			exclusions[exclusion] = true
			logWriter.Printf("Exclusion %s added as %s", entry, exclusion)
		} else {
			logWriter.Fatalf("File %s in exclusions does not exist or is inaccessible", entry)
			return
		}
	}

	if err = scanner.Err(); err != nil {
		logWriter.Fatalf("Error scanning file %s: %v", path, err)
	}

	return
}
