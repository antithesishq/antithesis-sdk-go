//go:build enable_antithesis_sdk

package instrumentation

import (
	"fmt"
	"sync/atomic"

	"github.com/antithesishq/antithesis-sdk-go/internal"
)

// Lease-word packing, from instrumentation.h ("Coverage leases"): these are
// part of the frozen v2 linkage ABI, so a hardcoded mirror is safe.
const (
	leaseEpochShift   = 40
	leaseGrantedShift = 20
	leaseFieldMask    = 0xFFFFF
	leaseEpochMask    = 0xFFFFFF
)

var defaultModule atomic.Pointer[Module]

const instrumentor_tag = "github.com/antithesishq/antithesis-sdk-go/instrumentation"

// InitializeModule should be called only once from a program.
func InitializeModule(symbolTable string, edgeCount int) uint64 {
	modulesMu.Lock()
	defer modulesMu.Unlock()
	if defaultModule.Load() != nil {
		panic("InitializeModule() has already been called!")
	}
	module := registerModuleLocked(symbolTable, edgeCount)
	defaultModule.Store(module)
	return module.offset
}

// Notify reports a hit through the module selected by InitializeModule.
func Notify(edge int) {
	module := defaultModule.Load()
	if module == nil {
		panic(fmt.Sprintf("%s.Notify() called before InitializeModule()", instrumentor_tag))
	}
	module.Notify(edge)
}

// newLeaseCache allocates per-edge lease storage for one coverage module, or
// returns nil when the loaded library predates leases — then every hit calls
// in, because the v1 return value's "never call again" is not revocable and
// caching it would go deaf to coverage-seen resets.
func newLeaseCache(edgeCount int) []uint64 {
	if _, ok := internal.LeaseGeneration(); !ok {
		return nil
	}
	// Some instrumentors use 1-based edge ids; index edgeCount is
	// reachable (the library sizes its guard table the same way).
	return make([]uint64, edgeCount+1)
}

// hasCachedLease reports whether edge has a lease word to count hits against.
// Anything else — a library predating leases, or an edge id outside the range
// the module registered — has to call in on every hit.
func hasCachedLease(edge int, leases []uint64) bool {
	return leases != nil && edge >= 0 && edge < len(leases)
}

// notifyEdge reports one hit of edge in a module based at offset, whose cached
// lease words are leases (nil if the library predates leases).
//
// Repeat hits are counted locally under a coverage lease (instrumentation.h,
// "Coverage leases") and reported when the lease runs out or is revoked — a
// revocation (e.g. a coverage-seen reset) reaches this cache through the epoch
// check, unlike the retired v1 never-call-again cache, which was deaf to it.
func notifyEdge(edge int, offset uint64, leases []uint64) {
	edgePlusOffset := uint64(edge) + offset
	if !hasCachedLease(edge, leases) {
		// v1 library (or an out-of-range edge): every hit calls in.
		internal.Notify(edgePlusOffset)
		return
	}
	lease, counted := leaseLocalHit(&leases[edge], internal.LeaseGeneration)
	if counted {
		return
	}
	// The store may clobber a concurrent local decrement of the fresh lease;
	// that only under-reports folded hits, which the ABI does tolerate (it
	// perturbs pause-interval statistics).
	if fresh, ok := internal.NotifyV2(edgePlusOffset, foldedHits(lease)); ok {
		atomic.StoreUint64(&leases[edge], fresh)
	}
}

// granted and remaining come from one atomic snapshot of the word, so
// granted >= remaining (the library's grant contract) and the
// subtraction cannot wrap.
func foldedHits(lease uint64) uint64 {
	granted := (lease >> leaseGrantedShift) & leaseFieldMask
	remaining := lease & leaseFieldMask
	return granted - remaining
}

// leaseLocalHit tries to count one hit locally against the cached lease
// word: it decrements the word only if, at the instant the decrement
// commits, the lease is still live — remaining nonzero and epoch current.
// The whole-word decrement is safe only under that CAS revalidation:
// remaining and granted are adjacent bitfields, so a decrement guarded by
// a stale snapshot can borrow across the field boundary, corrupt the word,
// and make a later call-in's granted-remaining wrap (which the library
// treats as fatal). Per instrumentation.h the caller decrements the
// *field*, not the word; this mirrors the library's own guard fast path
// (instrumentation/coverage.cpp), where compare_exchange re-tests both
// conditions against the current word on every iteration.
//
// Returns the last word observed and whether the hit was counted locally;
// when false, the caller must call in, folding that word's
// granted-remaining locally-counted hits.
func leaseLocalHit(lease *uint64, currentEpoch func() (uint64, bool)) (uint64, bool) {
	word := atomic.LoadUint64(lease)
	for {
		if word&leaseFieldMask == 0 {
			return word, false
		}
		// Re-read the revocation epoch every iteration; the header requires
		// the read to be non-hoistable out of hot loops.
		generation, ok := currentEpoch()
		if !ok || (word>>leaseEpochShift)&leaseEpochMask != generation&leaseEpochMask {
			return word, false
		}
		if atomic.CompareAndSwapUint64(lease, word, word-1) {
			return word - 1, true
		}
		word = atomic.LoadUint64(lease)
	}
}
