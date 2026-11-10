//go:build enable_antithesis_sdk

package instrumentation

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestRepeatedRegistrationSharesLeases(t *testing.T) {
	first := RegisterModule(t.Name(), 2)
	// The test runs without libvoidstar, so supply the lease it would grant.
	first.leases = []uint64{0, makeLease(2, 2), 0}
	second := RegisterModule(t.Name(), 2)
	if second != first {
		t.Fatal("repeated registration returned a different Module")
	}
	currentEpoch := func() (uint64, bool) { return 2, true }
	for _, module := range []*Module{first, second} {
		if _, counted := leaseLocalHit(&module.leases[1], currentEpoch); !counted {
			t.Fatal("shared lease failed to count a hit")
		}
	}
	if _, counted := leaseLocalHit(&first.leases[1], currentEpoch); counted {
		t.Fatal("registrations spent more than the shared grant")
	}
}

func TestConcurrentRegistrationSharesModule(t *testing.T) {
	const callers = 32
	start := make(chan struct{})
	results := make(chan *Module, callers)
	for range callers {
		go func() {
			<-start
			results <- RegisterModule(t.Name(), 4)
		}()
	}
	close(start)
	first := <-results
	for range callers - 1 {
		if module := <-results; module != first {
			t.Error("concurrent registration returned a different Module")
		}
	}
}

func TestDistinctCatalogsHaveDistinctModules(t *testing.T) {
	first := RegisterModule(t.Name()+"/first", 4)
	second := RegisterModule(t.Name()+"/second", 4)
	if first == second {
		t.Fatal("different catalogs returned the same Module")
	}
}

func TestRegistrationRejectsConflictingEdgeCountWithoutLeases(t *testing.T) {
	module := RegisterModule(t.Name(), 4)
	if module.leases != nil {
		t.Fatal("test requires unavailable lease support")
	}
	defer func() {
		if recover() == nil {
			t.Error("registration accepted a conflicting edge count")
		}
		if got := RegisterModule(t.Name(), 4); got != module {
			t.Error("rejected registration replaced the original Module")
		}
	}()
	RegisterModule(t.Name(), 5)
}

func TestRegistrationRejectsNegativeEdgeCount(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("registration accepted a negative edge count")
		}
	}()
	RegisterModule(t.Name(), -1)
}

func TestInitializeModuleSharesRegistration(t *testing.T) {
	for _, initializeFirst := range []bool{true, false} {
		name := "RegisterFirst"
		if initializeFirst {
			name = "InitializeFirst"
		}
		t.Run(name, func(t *testing.T) {
			previous := defaultModule.Swap(nil)
			t.Cleanup(func() { defaultModule.Store(previous) })
			var module *Module
			var offset uint64
			if initializeFirst {
				offset = InitializeModule(t.Name(), 4)
				module = RegisterModule(t.Name(), 4)
			} else {
				module = RegisterModule(t.Name(), 4)
				offset = InitializeModule(t.Name(), 4)
			}
			if defaultModule.Load() != module {
				t.Fatal("InitializeModule and RegisterModule use different Modules")
			}
			if offset != module.offset {
				t.Fatalf("InitializeModule returned offset %d, want %d", offset, module.offset)
			}
			Notify(1)
		})
	}
}

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
