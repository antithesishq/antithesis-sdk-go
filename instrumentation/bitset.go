package instrumentation

import (
	"math/bits"
	"sync"
)

// bitSet is a rudimentary implementation suitable for
// the Antithesis Go instrumentation wrappers. One
// can set bits; one cannot unset them. It is 0-indexed,
// although our edges begin at 1. The value type is
// int, to be consistent with our "edge" type. Negative
// index values will result in a panic.
type bitSet struct {
	// There seems to be no performance differences
	// among the different integer types.
	slots []uint64
	mutex sync.RWMutex
}

func (b *bitSet) slotAndBit(index int) (int, int) {
	// One assumes that the compiler will optimize these.
	return index / 64, index % 64
}

func (b *bitSet) get(slot, bit int) bool {
	if slot >= len(b.slots) {
		return false
	}
	mask := (uint64(1) << bit)
	return (b.slots[slot] & mask) != 0
}

// Get returns the value at this index.
func (b *bitSet) Get(index int) bool {
	slot, bit := b.slotAndBit(index)
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.get(slot, bit)
}

// Set will only switch a bit on.
func (b *bitSet) Set(index int) {
	slot, bit := b.slotAndBit(index)
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if slot >= len(b.slots) {
		// Go takes care of the *capacity* under the covers.
		// So we don't need a tricky implementation, or
		// a fixed size. Expansion is cheap.
		extension := 1 + int(slot) - len(b.slots)
		b.slots = append(b.slots, make([]uint64, extension)...)
	}
	mask := (uint64(1) << bit)
	b.slots[slot] |= mask
}
