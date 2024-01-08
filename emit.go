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
 // typedef void (*go_fuzz_error_message_fn)(const char *message);
 //
 // typedef const char * (*go_version_fn)();
 // const char *
 // go_version(void *f) {
 //   return ((go_version_fn)f)();
 // }
 //
 // typedef void (*go_fuzz_json_data_fn)(const char *data, size_t size);
 // void
 // go_fuzz_json_data(void *f, const char *data, size_t size) {
 //   ((go_fuzz_json_data_fn)f)(data, size);
 // }
 //
 // typedef void (*go_fuzz_set_source_name_fn)(const char *name);
 // void
 // go_fuzz_set_source_name(void *f, const char *name) {
 //   ((go_fuzz_set_source_name_fn)f)(name);
 // }
 //
 // typedef void (*go_fuzz_info_message_fn)(const char *message);
 // void
 // go_fuzz_info_message(void *f, const char *message) {
 //   ((go_fuzz_info_message_fn)f)(message);
 // }
 //
 // typedef void (*go_fuzz_error_message_fn)(const char *message);
 // void
 // go_fuzz_error_message(void *f, const char *message) {
 //   ((go_fuzz_error_message_fn)f)(message);
 // }
 import "C"

 type EmitInfo struct {
   dl_handle unsafe.Pointer
   json_data_handle unsafe.Pointer
   set_source_name_handle unsafe.Pointer
   info_message_handle unsafe.Pointer
   error_message_handle unsafe.Pointer
   out_f *os.File
 }

 var emitter = EmitInfo {
   dl_handle: nil,
   json_data_handle: nil,
   set_source_name_handle: nil,
   info_message_handle: nil,
   error_message_handle: nil,
   out_f: nil,
 }

 func JSONData(payload string) {
   if emitter.dl_handle != nil {
     nbx := len(payload)
     cstr_payload := C.CString(payload)
     C.go_fuzz_json_data(emitter.json_data_handle, cstr_payload, C.ulong(nbx))
     C.free(unsafe.Pointer(cstr_payload))
     return
   }
   if emitter.out_f != nil {
       emitter.out_f.WriteString(payload)
       emitter.out_f.WriteString("\n")
   }
 }

 func InfoMessage(message string) {
   if emitter.dl_handle != nil {
     cstr_message := C.CString(message)
     C.go_fuzz_info_message(emitter.info_message_handle, cstr_message)
     C.free(unsafe.Pointer(cstr_message))
     return
   }
   if emitter.out_f != nil {
       emitter.out_f.WriteString(message)
       emitter.out_f.WriteString("\n")
   }
 }

 func ErrorMessage(message string) {
   if emitter.dl_handle != nil {
     cstr_message := C.CString(message)
     C.go_fuzz_error_message(emitter.error_message_handle, cstr_message)
     C.free(unsafe.Pointer(cstr_message))
     return
   }
   if emitter.out_f != nil {
       emitter.out_f.WriteString(message)
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
        emitter.dl_handle = dl_handle
        event_logger_error("Can not connect to event logger")
        return false
     }

     // Send JSON
     var cstr_func_name *C.char

     cstr_func_name = C.CString("fuzz_json_data")
     var json_data_handle unsafe.Pointer = C.dlsym(dl_handle, cstr_func_name)
     C.free(unsafe.Pointer(cstr_func_name))
     if json_data_handle == nil {
        event_logger_error("Can not access fuzz_json_data")
     }

     // Set the source name
     cstr_func_name = C.CString("fuzz_set_source_name")
     var set_source_name_handle unsafe.Pointer = C.dlsym(dl_handle, cstr_func_name)
     C.free(unsafe.Pointer(cstr_func_name))
     if set_source_name_handle == nil {
        event_logger_error("Can not access fuzz_set_source_name")
     }
     
     // Send info message (stdout)
     cstr_func_name = C.CString("fuzz_info_message")
     var info_message_handle unsafe.Pointer = C.dlsym(dl_handle, cstr_func_name)
     C.free(unsafe.Pointer(cstr_func_name))
     if info_message_handle == nil {
        event_logger_error("Can not access fuzz_info_message")
     }
     
     // Send error message (stdout)
     cstr_func_name = C.CString("fuzz_error_message")
     var error_message_handle unsafe.Pointer = C.dlsym(dl_handle, cstr_func_name)
     C.free(unsafe.Pointer(cstr_func_name))
     if error_message_handle == nil {
        event_logger_error("Can not access fuzz_error_message")
     }
     
     // Save all handles for later dispatch
     emitter.dl_handle = dl_handle
     emitter.json_data_handle = json_data_handle
     emitter.set_source_name_handle = set_source_name_handle
     emitter.info_message_handle = info_message_handle
     emitter.error_message_handle = error_message_handle
     return dl_handle != nil
 }

 func close_shared_lib() {
   if emitter.dl_handle != nil {
     C.dlclose(emitter.dl_handle)
     emitter.dl_handle = nil
   }
   emitter.json_data_handle = nil
   emitter.set_source_name_handle = nil
   emitter.info_message_handle = nil
   emitter.error_message_handle = nil
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
