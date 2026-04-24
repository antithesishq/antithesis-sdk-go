package common

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

type Verbosity int

const (
	Normal Verbosity = 0
	Info  Verbosity = 1
	Debug Verbosity = 2
	Trace Verbosity = 3
)

type LogWriter struct {
	logger    *log.Logger
	verbosity Verbosity
}

// var logger *log.Logger
// var verbosity int = 0
var Logger *LogWriter

func NewLogWriter(logfileName string, vLevel Verbosity) {
	if Logger != nil {
		return
	}

	var erx error
	var fp *os.File

	wrx := os.Stderr
	logfilePath := strings.TrimSpace(logfileName)
	if logfilePath != "" {
		if fp, erx = os.Create(logfilePath); erx == nil {
			wrx = fp
		}
	}

	// Setting up the globals
	logger := log.New(wrx, "", log.LstdFlags|log.Lshortfile)
	verbosity := vLevel

	// Advise if the requested logfile was not created
	if erx != nil {
		logger.Printf("WARNING Unable to Create/Open requested logfile: %q", logfilePath)
	}

	Logger = &LogWriter{logger, verbosity}
}

func (lW *LogWriter) IsVerbose() bool {
	return (lW.verbosity > 0)
}

func (lW *LogWriter) VerboseLevel(v Verbosity) bool {
	return (v <= lW.verbosity)
}

func (lW *LogWriter) Printf(level Verbosity, format string, v ...any) {
	if level <= lW.verbosity {
		lW.logger.Printf(format, v...)
	}
}

func (lW *LogWriter) Fatal(v ...any) {
	lW.logger.Fatal(v...)
}

func (lW *LogWriter) Fatalf(format string, v ...any) {
	lW.logger.Fatalf(format, v...)
}
