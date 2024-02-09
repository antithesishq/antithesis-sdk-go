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

func (b *bitSet) _slotAndBit(index int) (int, int) {
	// One assumes that the compiler will optimize these.
	return index / 64, index % 64
}

func (b *bitSet) _get(slot, bit int) bool {
	if slot >= len(b.slots) {
		return false
	}
	mask := (uint64(1) << bit)
	return (b.slots[slot] & mask) != 0
}

// Size is needed only for unit tests; it's not
// required to be fast.
func (b *bitSet) Size() int {
	result := 0
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	for _, slot := range b.slots {
		result += bits.OnesCount64(slot)
	}
	return result
}

// Get returns the value at this index.
func (b *bitSet) Get(index int) bool {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	slot, bit := b._slotAndBit(index)
	return b._get(slot, bit)
}

// Set will only switch a bit on.
func (b *bitSet) Set(index int) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	slot, bit := b._slotAndBit(index)
	if slot >= len(b.slots) {
		// Go takes care of the *capacity* under the covers.
		// So we don't need a tricky implementation, or
		// a fixed size. Expansion is cheap.
		extension := 1+int(slot)-len(b.slots)
		b.slots = append(b.slots, make([]uint64, extension)...)
	}
	mask := (uint64(1) << bit)
	b.slots[slot] |= mask
}
