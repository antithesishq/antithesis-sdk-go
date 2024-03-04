package instrumentation

import (
	"fmt"
	"github.com/antithesishq/antithesis-sdk-go/assert"
	"github.com/antithesishq/antithesis-sdk-go/internal"
	"os"
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

	executable, _ := os.Executable()
	details := map[string]any{
		"executable":  executable,
		"symbolTable": symbolTable,
		"edgeCount":   edgeCount,
	}
	assert.Reachable("init_coverage_module() invoked", details)

	// WARN Re: integer type conversion, see https://github.com/golang/go/issues/29878
	offset := internal.InitCoverage(uint64(edgeCount), symbolTable)
	moduleOffset = uint64(offset)
	moduleInitialized = true
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
