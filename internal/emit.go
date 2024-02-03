package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"unsafe"
)

// --------------------------------------------------------------------------------
// To build and run an executable with this package
//
// CC=clang CGO_ENABLED=1 go run ./main.go
// --------------------------------------------------------------------------------

// #cgo LDFLAGS: -ldl
//
// #include <dlfcn.h>
// #include <stdbool.h>
// #include <stdint.h>
// #include <stdlib.h>
//
// typedef void (*go_fuzz_json_data_fn)(const char *data, size_t size);
// void
// go_fuzz_json_data(void *f, const char *data, size_t size) {
//   ((go_fuzz_json_data_fn)f)(data, size);
// }
//
// typedef void (*go_fuzz_flush_fn)(void);
// void
// go_fuzz_flush(void *f) {
//   ((go_fuzz_flush_fn)f)();
// }
//
// typedef uint64_t (*go_fuzz_get_random_fn)(void);
// uint64_t
// go_fuzz_get_random(void *f) {
//   return ((go_fuzz_get_random_fn)f)();
// }
//
import "C"

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

type libHandler interface {
	output(message string)
	random() uint64
}

const localOutputEnvVar = "ANTITHESIS_SDK_LOCAL_OUTPUT"
const errorLogLinePrefix = "[* antithesis-sdk-go *]"
const defaultNativeLibraryPath = "/usr/lib/libvoidstar.so"

var handler libHandler

type voidstarHandler struct {
	fuzzJsonData  unsafe.Pointer
	fuzzFlush     unsafe.Pointer
	fuzzGetRandom unsafe.Pointer
}

func (h *voidstarHandler) output(message string) {
	cstrMessage := C.CString(message)
	defer C.free(unsafe.Pointer(cstrMessage))
	C.go_fuzz_json_data(h.fuzzJsonData, cstrMessage, C.ulong(len(message)))
	C.go_fuzz_flush(h.fuzzFlush)
}

func (h *voidstarHandler) random() uint64 {
	return uint64(C.go_fuzz_get_random(h.fuzzGetRandom))
}

type localHandler struct {
	outputFile *os.File // can be nil
}

func (h *localHandler) output(message string) {
	if h.outputFile != nil {
		h.outputFile.WriteString(message + "\n")
	}
}

func (h *localHandler) random() uint64 {
	return rand.Uint64()
}

// If we have a file at `defaultNativeLibraryPath`, we load the shared library
// (and panic on any error encountered during load).
// Otherwise fallback to the local handler.
func init() {
	if _, err := os.Stat(defaultNativeLibraryPath); err == nil {
		if handler, err = openSharedLib(defaultNativeLibraryPath); err != nil {
			panic(err)
		}
		return
	}
	handler = openLocalHandler()
}

// Attempt to load libvoidstar and some symbols from `path`
func openSharedLib(path string) (*voidstarHandler, error) {
	cstrPath := C.CString(path)
	defer C.free(unsafe.Pointer(cstrPath))

	dlError := func(message string) error {
		return fmt.Errorf("%s: (%s)", message, C.GoString(C.dlerror()))
	}

	sharedLib := C.dlopen(cstrPath, C.int(C.RTLD_NOW))
	if sharedLib == nil {
		return nil, dlError("Can not load the Antithesis native library")
	}

	loadFunc := func(name string) (symbol unsafe.Pointer, err error) {
		cstrName := C.CString(name)
		defer C.free(unsafe.Pointer(cstrName))
		if symbol = C.dlsym(sharedLib, cstrName); symbol == nil {
			err = dlError(fmt.Sprintf("Can not access symbol %s", name))
		}
		return
	}

	fuzzJsonData, err := loadFunc("fuzz_json_data")
	if err != nil {
		return nil, err
	}
	fuzzFlush, err := loadFunc("fuzz_flush")
	if err != nil {
		return nil, err
	}
	fuzzGetRandom, err := loadFunc("fuzz_get_random")
	if err != nil {
		return nil, err
	}

	return &voidstarHandler{fuzzJsonData, fuzzFlush, fuzzGetRandom}, nil
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
