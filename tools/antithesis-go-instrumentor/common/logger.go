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

type LogWriter struct {
  verbosity int
  logger *log.Logger
}

// var logger *log.Logger
// var verbosity int = 0
var logWriter *LogWriter

func NewLogWriter(logfileName string, vLevel int) *LogWriter {
  if logWriter != nil {
    return logWriter
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
  
  logWriter = &LogWriter{verbosity, logger}
  return logWriter
}

func GetLogWriter() *LogWriter {
  return NewLogWriter("", 0);
}

func (lW *LogWriter)IsVerbose() bool {
	return (lW.verbosity > 0)
}

func (lW *LogWriter)VerboseLevel(v int) bool {
	return (v <= lW.verbosity)
}

func (lW *LogWriter)Printf(format string, v ...any) {
  lW.logger.Printf(format, v...)
}

func (lW *LogWriter)Fatal(v ...any) {
  lW.logger.Fatal(v...)
}

func (lW *LogWriter)Fatalf(format string, v ...any) {
  lW.logger.Fatalf(format, v...)
}
