package internal

import (
  "errors"
  "fmt"
  "os"
  "unsafe"
  // [PH] "unicode/utf8"
)

 // --------------------------------------------------------------------------------
 // To build and run an executable with this package
 //
 // CC=clang CGO_ENABLED=1 go run ./main.go haj
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
 //
 // typedef int (*go_fuzz_getchar_fn)(void);
 // int
 // go_fuzz_getchar(void *f) {
 //   return ((go_fuzz_getchar_fn)f)();
 // }
 //
 // // [PH] typedef int (*go_fuzz_putchar_fn)(char c);
 // // [PH] int
 // // [PH] go_fuzz_putchar(void *f, char c) {
 // // [PH]   return ((go_fuzz_putchar_fn)f)(c);
 // // [PH] }
 //
 // // [PH] typedef void (*go_fuzz_flush_fn)(void);
 // // [PH] void
 // // [PH] go_fuzz_flush(void *f) {
 // // [PH]   ((go_fuzz_flush_fn)f)();
 // // [PH] }
 //
 // typedef uint64_t (*go_fuzz_get_random_fn)(void);
 // uint64_t
 // go_fuzz_get_random(void *f) {
 //   return ((go_fuzz_get_random_fn)f)();
 // }
 //
 // typedef bool (*go_fuzz_coin_flip_fn)(void);
 // bool
 // go_fuzz_coin_flip(void *f) {
 //   return ((go_fuzz_coin_flip_fn)f)();
 // }
 import "C"

 type EmitInfo struct {
   dso_handle unsafe.Pointer
   json_data_handle unsafe.Pointer
   set_source_name_handle unsafe.Pointer
   info_message_handle unsafe.Pointer
   error_message_handle unsafe.Pointer
   getchar_handle unsafe.Pointer
   // [PH] putchar_handle unsafe.Pointer
   // [PH] flush_handle unsafe.Pointer
   get_random_handle unsafe.Pointer
   coin_flip_handle unsafe.Pointer
 }

 var emitter = EmitInfo {
   dso_handle: nil,
   json_data_handle: nil,
   set_source_name_handle: nil,
   info_message_handle: nil,
   error_message_handle: nil,
   getchar_handle: nil,
   // [PH] putchar_handle: nil,
   // [PH] flush_handle: nil,
   get_random_handle: nil,
   coin_flip_handle: nil,
 }

 var DSOError error = errors.New("No DSO Available")

 func Json_data(payload string) error {
   if emitter.dso_handle == nil {
       return DSOError
   }
   nbx := len(payload)
   cstr_payload := C.CString(payload)
   C.go_fuzz_json_data(emitter.json_data_handle, cstr_payload, C.ulong(nbx))
   C.free(unsafe.Pointer(cstr_payload))
   return nil
 }

 func Info_message(message string) error {
   if emitter.dso_handle == nil {
       return DSOError
   }
   cstr_message := C.CString(message)
   C.go_fuzz_info_message(emitter.info_message_handle, cstr_message)
   C.free(unsafe.Pointer(cstr_message))
   return nil
 }

 func Error_message(message string) error {
   if emitter.dso_handle == nil {
       return DSOError
   }
   cstr_message := C.CString(message)
   C.go_fuzz_error_message(emitter.error_message_handle, cstr_message)
   C.free(unsafe.Pointer(cstr_message))
   return nil
 }

 func Set_source_name(name string) error {
   if emitter.dso_handle == nil {
       return DSOError
   }
   cstr_name := C.CString(name)
   C.go_fuzz_set_source_name(emitter.set_source_name_handle, cstr_name)
   C.free(unsafe.Pointer(cstr_name))
   return nil
 }

 func Getchar() (r rune, err error) {
   if emitter.dso_handle == nil {
       return 0, DSOError
   }
   retval := C.go_fuzz_getchar(emitter.getchar_handle)
   return rune(retval), nil
 }

 // [PH] func Putchar(r rune) (r2 rune, err error) {
 // [PH]   if emitter.dso_handle == nil {
 // [PH]       return 0, DSOError
 // [PH]   }
 // [PH]   var retval C.int

 // [PH]   if utf8.RuneLen(r) == 1 {
 // [PH]       c := uint8(r)
 // [PH]      retval = C.go_fuzz_putchar(emitter.putchar_handle, C.char(c))
 // [PH]   }
 // [PH]   return rune(retval), nil
 // [PH] }

 // [PH] func Flush() error {
 // [PH]   if emitter.dso_handle == nil {
 // [PH]       return DSOError
 // [PH]   }
 // [PH]   C.go_fuzz_flush(emitter.flush_handle)
 // [PH]   return nil
 // [PH] }

 func Get_random() (v uint64, err error) {
   if emitter.dso_handle == nil {
       return 0, DSOError
   }
   retval := C.go_fuzz_get_random(emitter.get_random_handle)
   return uint64(retval), nil
 }

 func Coin_flip() (b bool, err error){
   if emitter.dso_handle == nil {
       return false, DSOError
   }
   retval := C.go_fuzz_coin_flip(emitter.coin_flip_handle)
   return bool(retval), nil
 }

 func try_shared_lib(lib_path string) bool {
   close_shared_lib()
   did_open := open_shared_lib(lib_path)
    if !did_open {
      open_failed_handler()
    }
    return did_open
 }

 func open_failed_handler() {
      fmt.Printf("\n    [* antithesis-sdk-go *] Will handle events locally ---\n\n")
 }


 // Open the target library
 func open_shared_lib(lib_path string) bool {
     loading_mode := C.int(C.RTLD_NOW)
     cstr_lib_path := C.CString(lib_path)

     var dso_handle unsafe.Pointer = C.dlopen(cstr_lib_path, loading_mode)
     C.free(unsafe.Pointer(cstr_lib_path))

     if dso_handle == nil {
        emitter.dso_handle = dso_handle
        event_logger_error("Can not connect to event logger")
        return false
     }

     // Send JSON
     var cstr_func_name *C.char

     cstr_func_name = C.CString("fuzz_json_data")
     var json_data_handle unsafe.Pointer = C.dlsym(dso_handle, cstr_func_name)
     C.free(unsafe.Pointer(cstr_func_name))
     if json_data_handle == nil {
        event_logger_error("Can not access fuzz_json_data")
     }

     // Set the source name
     cstr_func_name = C.CString("fuzz_set_source_name")
     var set_source_name_handle unsafe.Pointer = C.dlsym(dso_handle, cstr_func_name)
     C.free(unsafe.Pointer(cstr_func_name))
     if set_source_name_handle == nil {
        event_logger_error("Can not access fuzz_set_source_name")
     }
     
     // Send info message (stdout)
     cstr_func_name = C.CString("fuzz_info_message")
     var info_message_handle unsafe.Pointer = C.dlsym(dso_handle, cstr_func_name)
     C.free(unsafe.Pointer(cstr_func_name))
     if info_message_handle == nil {
        event_logger_error("Can not access fuzz_info_message")
     }
     
     // Send error message (stdout)
     cstr_func_name = C.CString("fuzz_error_message")
     var error_message_handle unsafe.Pointer = C.dlsym(dso_handle, cstr_func_name)
     C.free(unsafe.Pointer(cstr_func_name))
     if error_message_handle == nil {
        event_logger_error("Can not access fuzz_error_message")
     }
     
     // Get a character
     cstr_func_name = C.CString("fuzz_getchar")
     var getchar_handle unsafe.Pointer = C.dlsym(dso_handle, cstr_func_name)
     C.free(unsafe.Pointer(cstr_func_name))
     if getchar_handle == nil {
        event_logger_error("Can not access fuzz_getchar")
     }
     
     // [PH] // Put a character
     // [PH] cstr_func_name = C.CString("fuzz_putchar")
     // [PH] var putchar_handle unsafe.Pointer = C.dlsym(dso_handle, cstr_func_name)
     // [PH] C.free(unsafe.Pointer(cstr_func_name))
     // [PH] if putchar_handle == nil {
     // [PH]    event_logger_error("Can not access fuzz_putchar")
     // [PH] }
     // [PH] 
     // [PH] // Flush pending output
     // [PH] cstr_func_name = C.CString("fuzz_flush")
     // [PH] var flush_handle unsafe.Pointer = C.dlsym(dso_handle, cstr_func_name)
     // [PH] C.free(unsafe.Pointer(cstr_func_name))
     // [PH] if flush_handle == nil {
     // [PH]    event_logger_error("Can not access fuzz_flush")
     // [PH] }

     // Get a random uint64
     cstr_func_name = C.CString("fuzz_get_random")
     var get_random_handle unsafe.Pointer = C.dlsym(dso_handle, cstr_func_name)
     C.free(unsafe.Pointer(cstr_func_name))
     if get_random_handle == nil {
        event_logger_error("Can not access fuzz_get_random")
     }

     // Get a coin flip bool
     cstr_func_name = C.CString("fuzz_coin_flip")
     var coin_flip_handle unsafe.Pointer = C.dlsym(dso_handle, cstr_func_name)
     C.free(unsafe.Pointer(cstr_func_name))
     if coin_flip_handle == nil {
        event_logger_error("Can not access fuzz_coin_flip")
     }
     
     // Save all handles for later dispatch
     emitter.dso_handle = dso_handle
     emitter.json_data_handle = json_data_handle
     emitter.set_source_name_handle = set_source_name_handle
     emitter.info_message_handle = info_message_handle
     emitter.error_message_handle = error_message_handle
     emitter.getchar_handle = getchar_handle
     // [PH] emitter.putchar_handle = putchar_handle
     // [PH] emitter.flush_handle = flush_handle
     emitter.get_random_handle = get_random_handle
     emitter.coin_flip_handle = coin_flip_handle
     return dso_handle != nil
 }

 func close_shared_lib() {
   if emitter.dso_handle != nil {
     C.dlclose(emitter.dso_handle)
     emitter.dso_handle = nil
   }
   emitter.json_data_handle = nil
   emitter.set_source_name_handle = nil
   emitter.info_message_handle = nil
   emitter.error_message_handle = nil
   emitter.getchar_handle = nil
   // [PH] emitter.putchar_handle = nil
   // [PH] emitter.flush_handle = nil
   emitter.get_random_handle = nil
   emitter.coin_flip_handle = nil
 }

 func event_logger_error(what string) {
   err_txt := C.GoString(C.dlerror())
   fmt.Fprintf(os.Stderr, "\n    [* antithesis-sdk-go *] %s =->  %s\n\n", what, err_txt)
 }

 func init() {
    var did_open bool = false

    // lib_path := os.Getenv("ANTILOG_PATH") // Use this DSO
    lib_path := "/usr/lib/libvoidstar.so"
    if len(lib_path) > 0 {
        if did_open = open_shared_lib(lib_path); did_open {
            return
        }
    }
 }
