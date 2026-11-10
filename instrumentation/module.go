//go:build enable_antithesis_sdk

package instrumentation

import (
	"fmt"
	"sync"

	"github.com/antithesishq/antithesis-sdk-go/internal"
)

// A Module is one independently instrumented unit of coverage, registered with
// its own symbol table and its own edge space.
type Module struct {
	// offset is what init_coverage_module() returned: the value that must be
	// added to this module's edge ids before they are passed to notify_coverage().
	offset    uint64
	edgeCount int

	// leases caches one coverage-lease word per edge, letting repeat hits be
	// counted locally without calling into the library; see notifyEdge. It is
	// nil when the loaded library predates leases.
	//
	// Each module holds its own leases because each has its own edge space,
	// numbered from 1. The revocation epoch they are validated against is
	// process-wide, so one revocation still voids every module's leases at once.
	leases []uint64
}

var (
	modulesMu        sync.Mutex
	modulesByCatalog = make(map[string]*Module)
)

// RegisterModule returns the shared coverage module for symbolTable. Repeated
// registrations must agree on edgeCount.
func RegisterModule(symbolTable string, edgeCount int) *Module {
	modulesMu.Lock()
	defer modulesMu.Unlock()
	return registerModuleLocked(symbolTable, edgeCount)
}

func registerModuleLocked(symbolTable string, edgeCount int) *Module {
	if edgeCount < 0 {
		panic("RegisterModule() requires a nonnegative edge count")
	}
	if module := modulesByCatalog[symbolTable]; module != nil {
		if module.edgeCount != edgeCount {
			panic(fmt.Sprintf("module %q registered with %d edges; previously registered with %d",
				symbolTable, edgeCount, module.edgeCount))
		}
		return module
	}
	// Copies of a catalog share native pause counters, so they must also share
	// the leases that allow hits to bypass those counters.
	module := &Module{
		offset:    internal.InitCoverage(uint64(edgeCount), symbolTable),
		edgeCount: edgeCount,
		leases:    newLeaseCache(edgeCount),
	}
	modulesByCatalog[symbolTable] = module
	return module
}

// Notify reports that edge was reached. edge is relative to this module and
// 0 <= edge <= edgeCount. Repeat hits are absorbed by this module's coverage
// leases
func (m *Module) Notify(edge int) {
	notifyEdge(edge, m.offset, m.leases)
}
