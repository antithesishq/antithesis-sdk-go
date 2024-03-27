package random

// Returns a randomly chosen item from a list of options. You should not store this value, but should use it immediately.
//
// This function is not purely for convenience. Signaling to the Antithesis platform that you intend to use a random value in a structured way enables it to provide more interesting choices over time.
func RandomChoice(things []any) any {
	num_things := len(things)
	if num_things < 1 {
		return nil
	}

	uval := GetRandom()
	index := uval % uint64(num_things)
	return things[index]
}
