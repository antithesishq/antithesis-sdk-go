//go:build !no_antithesis_sdk

package internal

import (
	"log"
	"math/rand"
	"os"
)

type localHandler struct {
	outputFile *os.File // can be nil
}

func (h *localHandler) output(message string) {
	msg_len := len(message)
	if msg_len == 0 {
		return
	}
	if h.outputFile != nil {
		h.outputFile.WriteString(message + "\n")
	}
}

func (h *localHandler) random() uint64 {
	return rand.Uint64()
}

func (h *localHandler) notify(edge uint64) bool {
	return false
}

func (h *localHandler) init_coverage(num_edges uint64, symbols string) uint64 {
	return 0
}

// If `localOutputEnvVar` is set to a non-empty path, attempt to open that path and truncate the file
// to serve as the log file of the local handler.
// Otherwise, we don't have a log file, and logging is a no-op in the local handler.
func openLocalHandler() *localHandler {
	path, is_set := os.LookupEnv(localOutputEnvVar)
	if !is_set || len(path) == 0 {
		return &localHandler{nil}
	}

	// Open the file R/W (create if needed and possible)
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Printf("%s Failed to open path %s: %v", errorLogLinePrefix, path, err)
		file = nil
	} else if err = file.Truncate(0); err != nil {
		log.Printf("%s Failed to truncate file at %s: %v", errorLogLinePrefix, path, err)
		file = nil
	}

	return &localHandler{file}
}
