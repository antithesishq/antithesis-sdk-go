package instrumentation

import (
	"fmt"
	"github.com/antithesishq/antithesis-sdk-go/internal"
	"os"
	"unsafe"
)

var moduleInitialized = false
var moduleOffset uint64
var edgesVisited = bitSet{}

const instrumentor_tag = "github.com/antithesishq/antithesis-sdk-go/instrumentation"

// InitializeModule should be called only once from a program.
func InitializeModule(symbolTable string, edgeCount int) uint64 {
	if moduleInitialized {
		// We cannot support incorrect workflows.
		panic("InitializeModule() has already been called!")
	}
	// WARN Re: integer type conversion, see https://github.com/golang/go/issues/29878
	executable, _ := os.Executable()
	// [PH] InfoMessage(fmt.Sprintf("%s called antithesis.com/go/instrumentation.InitializeModule(%s, %d)", executable, symbolTable, edgeCount))
	msg := fmt.Sprintf("%s called %s.InitializeModule(%s, %d)", executable, instrumentor_tag, symbolTable, edgeCount)
	internal.Json_data(msg)

	offset := internal.InitCoverage(edgeCount, symbolTable)
	// [PH] s := C.CString(symbolTable)
	// [PH] defer C.free(unsafe.Pointer(s))
	// [PH] offset := C.init_coverage_module(C.ulong(edgeCount), s)
	moduleOffset = uint64(offset)
	moduleInitialized = true
	// TODO Determine if any custom code would ever want this; return void if not.
	return moduleOffset
}

// Notify will be called from instrumented code.
func Notify(edge int) {
	if !moduleInitialized {
		// We cannot support incorrect workflows.
		panic(fmt.Sprintf("%s.Notify() called before InitializeModule()", instrumentor_tag))
	}
	if edgesVisited.Get(edge) {
		return
	}
	edgePlusOffset := uint64(edge)
	edgePlusOffset += moduleOffset
	mustCall := internal.Notify(edgePlusOffset)
	if !mustCall {
		edgesVisited.Set(edge)
	}
}

// FuzzExit is inserted by the instrumentation.
func FuzzExit(exit int) {
	return
	// internal.Fuzz_exit(C.int(exit))
}