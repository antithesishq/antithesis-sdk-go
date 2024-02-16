package main

import (
	"log"
	"os"
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

func VerboseLevel(v int) bool {
	return (v <= verbosity)
}

func CreateGlobalLogger(logfileName string, vLevel int) {
	wrx := os.Stderr
	logfilePath := strings.TrimSpace(logfileName)
	if logfilePath != "" {
		if fp, erx := os.Create(logfilePath); erx == nil {
			wrx = fp
		}
	}

	// Setting up the globals
	logger = log.New(wrx, "", log.LstdFlags|log.Lshortfile)
	verbosity = vLevel
}
