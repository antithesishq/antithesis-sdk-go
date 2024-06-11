//go:build !no_antithesis_sdk

package internal

import (
	"encoding/json"
	"errors"
	"os"
)

func Json_data(v any) error {
	if data, err := json.Marshal(v); err != nil {
		return err
	} else {
		handler.output(string(data))
		return nil
	}
}

func Get_random() uint64 {
	return handler.random()
}

func Notify(edge uint64) bool {
	return handler.notify(edge)
}

func InitCoverage(num_edges uint64, symbols string) uint64 {
	return handler.init_coverage(num_edges, symbols)
}

type libHandler interface {
	output(message string)
	random() uint64
	notify(edge uint64) bool
	init_coverage(num_edges uint64, symbols string) uint64
}

const (
	errorLogLinePrefix       = "[* antithesis-sdk-go *]"
	defaultNativeLibraryPath = "/usr/lib/libvoidstar.so"
	useLocalHandler          = "use-local"
)

var handler libHandler

// If we have a file at `defaultNativeLibraryPath`, we load the shared library
// (and panic on any error encountered during load).
// Otherwise fallback to the local handler.
func init() {
	var maybe_handler interface{}
	if _, err := os.Stat(defaultNativeLibraryPath); err == nil {
		if maybe_handler, err = openSharedLib(defaultNativeLibraryPath); err != nil {
			if err.Error() != useLocalHandler {
				panic(err)
			} else {
				handler = openLocalHandler()
				return
			}
		}

		// successfully opened the shared lib and can use it as intended
		// just ensure that it supports the libHandler interface
		ok := false
		handler, ok = maybe_handler.(libHandler)
		if !ok {
			panic(errors.New("unsupported library"))
		}
		return
	}
	handler = openLocalHandler()
}
