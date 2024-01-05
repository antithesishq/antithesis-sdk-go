package antilog

import (
  "fmt"
  "os"
  "unsafe"
)

 // --------------------------------------------------------------------------------
 // To build and run an executable with this package
 //
 // CC=clang CGO_ENABLED=1 go run ./main.go haj
 // --------------------------------------------------------------------------------


 // #cgo LDFLAGS: -ldl
 //
 // #include <dlfcn.h>
 // #include <stdlib.h>
 //
 // typedef const char * (*go_version_fn)();
 // typedef void (*go_fuzz_json_data_fn)(const char *data, size_t size);
 //
 // const char *
 // go_version(void *f) {
 //   return ((go_version_fn)f)();
 // }
 //
 // void
 // go_fuzz_json_data(void *f, const char *data, size_t size) {
 //   ((go_fuzz_json_data_fn)f)(data, size);
 // }
 import "C"

 type EmitInfo struct {
   dl_handle unsafe.Pointer
   emit_handle unsafe.Pointer
   out_f *os.File
 }

 var emitter = EmitInfo {
   dl_handle: nil,
   emit_handle: nil,
   out_f: nil,
 }

 func Emit(payload string) {
   if emitter.dl_handle != nil {
     nbx := len(payload)
     cstr_payload := C.CString(payload)
     C.go_fuzz_json_data(emitter.emit_handle, cstr_payload, C.ulong(nbx))
     C.free(unsafe.Pointer(cstr_payload))
     return
   }
   if emitter.out_f != nil {
       emitter.out_f.WriteString(payload)
       emitter.out_f.WriteString("\n")
   }
 }

 func TrySharedLib(lib_path string) bool {
   close_shared_lib()
   did_open := open_shared_lib(lib_path)
    if !did_open {
      open_failed_handler()
    }
    if did_open {
        close_output_file()
    }
    return did_open
 }

 func open_failed_handler() {
      fmt.Printf("\n    [* antilog *] Will handle events locally ---\n\n")
 }


 // Open the target library
 func open_shared_lib(lib_path string) bool {
     loading_mode := C.int(C.RTLD_NOW)
     cstr_lib_path := C.CString(lib_path)

     var dl_handle unsafe.Pointer = C.dlopen(cstr_lib_path, loading_mode)
     C.free(unsafe.Pointer(cstr_lib_path))

     if dl_handle == nil {
        event_logger_error("Can not connect to event logger")
        return false
     }

     // Send some JSON to the event capture
     cstr_func_name := C.CString("fuzz_json_data")
     var emit_handle unsafe.Pointer = C.dlsym(dl_handle, cstr_func_name)
     C.free(unsafe.Pointer(cstr_func_name))
     if emit_handle == nil {
        event_logger_error("Can not access fuzz_json_data()")
        C.dlclose(dl_handle)
     }
     
     emitter.dl_handle = dl_handle
     emitter.emit_handle = emit_handle
     return dl_handle != nil
 }

 func close_shared_lib() {
   if emitter.dl_handle != nil {
     C.dlclose(emitter.dl_handle)
     emitter.dl_handle = nil
   }
   emitter.emit_handle = nil
 }

 func open_output_file(out_path string) bool {
  var file *os.File
  var err error
  if file, err = os.OpenFile(out_path, os.O_RDWR | os.O_CREATE, 0644); err != nil {
    file = nil
  }
  if file != nil {
    if err = file.Truncate(0); err != nil {
      file = nil
    }
  }
  if file != nil {
    emitter.out_f = file
  }
  return err == nil
 }

 func close_output_file() {
     if emitter.out_f != nil {
         emitter.out_f.Close()
         emitter.out_f = nil
     }
 }

 func event_logger_error(what string) {
   err_txt := C.GoString(C.dlerror())
   fmt.Fprintf(os.Stderr, "\n    [* antilog *] %s =->  %s\n\n", what, err_txt)
 }

 func init() {
    var did_open bool = false

    lib_path := os.Getenv("ANTILOG_PATH") // Use this DSO
    if len(lib_path) > 0 {
        if did_open = open_shared_lib(lib_path); did_open {
            return
        }
    }

    out_path := os.Getenv("ANTILOG_OUTPUT") // Write output to this file
    if (len(out_path) > 0) {
        if did_open = open_output_file(out_path); did_open {
            return
        }
        fmt.Fprintf(os.Stderr, "\n    [* antilog *] Unable to open %q\n", out_path)
    }
 }
