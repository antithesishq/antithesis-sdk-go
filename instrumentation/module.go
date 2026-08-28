//go:build enable_antithesis_sdk

package instrumentation

import (
	"github.com/antithesishq/antithesis-sdk-go/internal"
)

// A Module is one independently instrumented unit of coverage, registered with
// its own symbol table and its own edge space.
type Module struct {
	// offset is what init_coverage_module() returned: the value that must be
	// added to this module's edge ids before they are passed to notify_coverage().
	offset uint64

	// leases caches one coverage-lease word per edge, letting repeat hits be
	// counted locally without calling into the library; see notifyEdge. It is
	// nil when the loaded library predates leases.
	//
	// Each module holds its own leases because each has its own edge space,
	// numbered from 1. The revocation epoch they are validated against is
	// process-wide, so one revocation still voids every module's leases at once.
	leases []uint64
}

// RegisterModule registers symbolTable as a new coverage module holding
// edgeCount edges, and returns a handle for reporting them.
func RegisterModule(symbolTable string, edgeCount int) *Module {
	// WARN Re: integer type conversion, see https://github.com/golang/go/issues/29878
	return &Module{
		offset: internal.InitCoverage(uint64(edgeCount), symbolTable),
		leases: newLeaseCache(edgeCount),
	}
}

// Notify reports that edge was reached. edge is relative to this module and
// 0 <= edge <= edgeCount. Repeat hits are absorbed by this module's coverage
// leases
func (m *Module) Notify(edge int) {
	notifyEdge(edge, m.offset, m.leases)
}
