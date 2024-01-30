package internal

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"time"
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

type emitInfo struct {
	dso_handle        unsafe.Pointer
	json_data_handle  unsafe.Pointer
	flush_handle      unsafe.Pointer
	get_random_handle unsafe.Pointer
}

var emitter = emitInfo{
	dso_handle:        nil,
	json_data_handle:  nil,
	flush_handle:      nil,
	get_random_handle: nil,
}

type localHandling struct {
	out_f         *os.File
	can_be_opened bool
	start_time    time.Time
}

var local_handler = &localHandling{
	out_f:         nil,
	can_be_opened: true,
	start_time:    time.Now(),
}

const localOutputEnvVar = "ANTITHESIS_SDK_LOCAL_OUTPUT"
const errorLogLinePrefix = "[* antithesis-sdk-go *]"
const defaultNativeLibraryPath = "/usr/lib/libvoidstar.so"

func no_emit() bool {
	if emitter.dso_handle == nil && !local_handler.can_be_opened {
		return true
	}
	if local_handler.out_f == nil {
		if len(os.Getenv(localOutputEnvVar)) == 0 {
			local_handler.can_be_opened = false
			return true
		}
	}
	return false
}

func Json_data(v any) error {
	if no_emit() {
		return nil
	}

	var data []byte = nil
	var err error
	if data, err = json.Marshal(v); err != nil {
		return err
	}
	payload := string(data)

	if emitter.dso_handle == nil {
		local_handler.emit(payload)
		return nil
	}

	nbx := len(payload)
	cstr_payload := C.CString(payload)
	C.go_fuzz_json_data(emitter.json_data_handle, cstr_payload, C.ulong(nbx))
	C.free(unsafe.Pointer(cstr_payload))
	flush()
	return nil
}

// TODO: we do not call this but I'm not sure that we should not be calling this after we call the json output functions
func flush() error {
	if emitter.dso_handle != nil {
		C.go_fuzz_flush(emitter.flush_handle)
	}
	return nil
}

func Get_random() uint64 {
	if emitter.dso_handle == nil {
		var err error
		var randInt *big.Int
		max := big.NewInt(math.MaxInt64)
		if randInt, err = rand.Int(rand.Reader, max); err != nil {
			panic(err)
		}
		return randInt.Uint64()
	}
	retval := C.go_fuzz_get_random(emitter.get_random_handle)
	return uint64(retval)
}

func open_failed_handler() {
	fmt.Printf("\n    %s Events will be handled locally ---\n\n", errorLogLinePrefix)
}

func exists_at_path(fullname string) (name string, exists bool) {
	exists = false
	name = fullname
	if _, err := os.Stat(fullname); err == nil {
		exists = true
		return
	}
	return
}

// Open the target library
func open_shared_lib(lib_path string) bool {
	if _, exists := exists_at_path(lib_path); !exists {
		return false
	}
	loading_mode := C.int(C.RTLD_NOW)
	cstr_lib_path := C.CString(lib_path)

	var dso_handle unsafe.Pointer = C.dlopen(cstr_lib_path, loading_mode)
	C.free(unsafe.Pointer(cstr_lib_path))

	if dso_handle == nil {
		emitter.dso_handle = dso_handle
		event_logger_error("Can not connect to Antithesis native library")
		return false
	}

	var cstr_func_name *C.char

	// Send JSON
	cstr_func_name = C.CString("fuzz_json_data")
	var json_data_handle unsafe.Pointer = C.dlsym(dso_handle, cstr_func_name)
	C.free(unsafe.Pointer(cstr_func_name))
	if json_data_handle == nil {
		event_logger_error("Can not access fuzz_json_data")
	}

	// Flush pending output
	cstr_func_name = C.CString("fuzz_flush")
	var flush_handle unsafe.Pointer = C.dlsym(dso_handle, cstr_func_name)
	C.free(unsafe.Pointer(cstr_func_name))
	if flush_handle == nil {
		event_logger_error("Can not access fuzz_flush")
	}

	// Get a random uint64
	cstr_func_name = C.CString("fuzz_get_random")
	var get_random_handle unsafe.Pointer = C.dlsym(dso_handle, cstr_func_name)
	C.free(unsafe.Pointer(cstr_func_name))
	if get_random_handle == nil {
		event_logger_error("Can not access fuzz_get_random")
	}

	// Save all handles for later dispatch
	emitter.dso_handle = dso_handle
	emitter.json_data_handle = json_data_handle
	emitter.flush_handle = flush_handle
	emitter.get_random_handle = get_random_handle
	return dso_handle != nil
}

func close_shared_lib() {
	if emitter.dso_handle != nil {
		C.dlclose(emitter.dso_handle)
		emitter.dso_handle = nil
	}
	emitter.json_data_handle = nil
	emitter.flush_handle = nil
	emitter.get_random_handle = nil
}

func event_logger_error(what string) {
	err_txt := C.GoString(C.dlerror())
	fmt.Fprintf(os.Stderr, "%s %s =->  %s\n\n", errorLogLinePrefix, what, err_txt)
}

func (pout *localHandling) open_output_file() error {
	var file *os.File
	var err error

	// Make sure we have user intent to open a file
	out_path := os.Getenv(localOutputEnvVar) // Write output to this file
	if len(out_path) == 0 {
		return errors.New("No local output")
	}

	// Open the file R/W (create if needed and possible)
	if file, err = os.OpenFile(out_path, os.O_RDWR|os.O_CREATE, 0644); err != nil {
		file = nil
		return err
	}

	// Truncate the file if possible (if not, consider the file unusable)
	if file != nil {
		if err = file.Truncate(0); err != nil {
			file = nil
			return err
		}
	}

	if file != nil {
		pout.out_f = file
	}

	return err
}

func (pout *localHandling) emit(payload string) {
	var err error
	if !pout.can_be_opened {
		return
	}
	if pout.out_f == nil {
		if err = pout.open_output_file(); err != nil {
			pout.can_be_opened = false
			return
		}
	}
	pout.out_f.WriteString(payload + "\n")
	return
}

func init() {
	var did_open bool = false
	lib_path := defaultNativeLibraryPath
	if len(lib_path) > 0 {
		if did_open = open_shared_lib(lib_path); did_open {
			return
		}
	}
}
