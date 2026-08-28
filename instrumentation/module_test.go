//go:build enable_antithesis_sdk

package instrumentation

import (
	"sync"
	"sync/atomic"
	"testing"
)

// Without a loaded library there are no leases to cache, and every hit must call
// in rather than being silently absorbed.
func TestRegisterModuleWithoutLeaseSupport(t *testing.T) {
	module := RegisterModule("go-test.sym.tsv", 8)
	if module.leases != nil {
		t.Fatalf("expected no lease cache without library support, got %d words", len(module.leases))
	}
	// Reaching the library-less path must not panic, at any edge.
	for _, edge := range []int{0, 1, 8, -1, 9999} {
		module.Notify(edge)
	}
}

// Each module keeps its own edge space, so concurrent traffic through two
// modules must not disturb the other's lease words. This exercises the per-module
// state that replaced the single-module globals.
func TestModulesDoNotShareLeaseState(t *testing.T) {
	const edges = 4
	currentEpoch := func() (uint64, bool) { return 2, true }

	first := &Module{offset: 1 << 32, leases: make([]uint64, edges+1)}
	second := &Module{offset: 2 << 32, leases: make([]uint64, edges+1)}
	for edge := 1; edge <= edges; edge++ {
		first.leases[edge] = makeLease(2, 3)
		second.leases[edge] = makeLease(2, 3)
	}

	var start, done sync.WaitGroup
	start.Add(1)
	done.Add(2 * edges)
	for edge := 1; edge <= edges; edge++ {
		for _, module := range []*Module{first, second} {
			go func(module *Module, edge int) {
				defer done.Done()
				start.Wait()
				for i := 0; i < 3; i++ {
					leaseLocalHit(&module.leases[edge], currentEpoch)
				}
			}(module, edge)
		}
	}
	start.Done()
	done.Wait()

	// Every lease had exactly its grant consumed, in both modules independently.
	for edge := 1; edge <= edges; edge++ {
		for name, module := range map[string]*Module{"first": first, "second": second} {
			epoch, granted, remaining := leaseFields(atomic.LoadUint64(&module.leases[edge]))
			if epoch != 2 || granted != 3 || remaining != 0 {
				t.Errorf("%s module edge %d: epoch=%d granted=%d remaining=%d, want 2/3/0",
					name, edge, epoch, granted, remaining)
			}
		}
	}
}
