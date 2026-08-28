//go:build !no_antithesis_sdk

package instrumentation

import (
	"sync"
	"sync/atomic"
	"testing"
)

func makeLease(epoch, grant uint64) uint64 {
	return (epoch&leaseEpochMask)<<leaseEpochShift | grant<<leaseGrantedShift | grant
}

func leaseFields(word uint64) (epoch, granted, remaining uint64) {
	return (word >> leaseEpochShift) & leaseEpochMask,
		(word >> leaseGrantedShift) & leaseFieldMask,
		word & leaseFieldMask
}

// Many goroutines hammer one lease word whose grant is smaller than the
// goroutine count. The lease may hand out at most `grant` local hits, and
// the decrements must stay inside the remaining field: a decrement guarded
// by a stale snapshot borrows into granted, which breaks the library's
// remaining <= granted invariant and makes the next call-in's
// granted-remaining wrap — a library-side fatal in production.
func TestLeaseLocalHitNeverOverdraws(t *testing.T) {
	const goroutines = 64
	currentEpoch := func() (uint64, bool) { return 7, true }
	for round := 0; round < 300; round++ {
		grant := uint64(1 + round%3)
		lease := makeLease(7, grant)
		var counted atomic.Uint64
		var start, done sync.WaitGroup
		start.Add(1)
		done.Add(goroutines)
		for g := 0; g < goroutines; g++ {
			go func() {
				defer done.Done()
				start.Wait()
				if _, ok := leaseLocalHit(&lease, currentEpoch); ok {
					counted.Add(1)
				}
			}()
		}
		start.Done()
		done.Wait()

		if counted.Load() > grant {
			t.Fatalf(
				"round %d: %d hits counted locally under a grant of %d",
				round, counted.Load(), grant,
			)
		}
		epoch, granted, remaining := leaseFields(atomic.LoadUint64(&lease))
		if epoch != 7 || granted != grant || remaining != grant-counted.Load() {
			t.Fatalf(
				"round %d: lease word corrupted: epoch=%d granted=%d remaining=%d (grant %d, %d counted)",
				round, epoch, granted, remaining, grant, counted.Load(),
			)
		}
	}
}

// The revocation race: goroutines that passed the epoch check against a
// stale snapshot must not decrement a word that was refreshed underneath
// them. The epoch flips mid-storm; whatever interleaving occurs, the word's
// fields must remain consistent and local hits must never exceed the grant.
func TestLeaseLocalHitStopsAtRevocation(t *testing.T) {
	const goroutines = 64
	for round := 0; round < 300; round++ {
		grant := uint64(1 + round%3)
		lease := makeLease(3, grant)
		var epochWord atomic.Uint64
		epochWord.Store(3)
		currentEpoch := func() (uint64, bool) { return epochWord.Load(), true }
		var counted atomic.Uint64
		var start, done sync.WaitGroup
		start.Add(1)
		done.Add(goroutines)
		for g := 0; g < goroutines; g++ {
			revoker := g == 0
			go func() {
				defer done.Done()
				start.Wait()
				if revoker {
					epochWord.Add(1)
					return
				}
				if _, ok := leaseLocalHit(&lease, currentEpoch); ok {
					counted.Add(1)
				}
			}()
		}
		start.Done()
		done.Wait()

		if counted.Load() > grant {
			t.Fatalf(
				"round %d: %d hits counted locally under a grant of %d",
				round, counted.Load(), grant,
			)
		}
		epoch, granted, remaining := leaseFields(atomic.LoadUint64(&lease))
		if epoch != 3 || granted != grant || remaining != grant-counted.Load() {
			t.Fatalf(
				"round %d: lease word corrupted: epoch=%d granted=%d remaining=%d (grant %d, %d counted)",
				round, epoch, granted, remaining, grant, counted.Load(),
			)
		}
	}
}
