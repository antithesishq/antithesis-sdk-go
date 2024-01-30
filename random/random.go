package random

import (
	"github.com/antithesishq/antithesis-sdk-go/internal"
)

func GetRandom() uint64 {
	return internal.Get_random()
}

func RandomChoice(things []any) any {
	num_things := len(things)
	if num_things < 1 {
		return nil
	}

	uval := GetRandom()
	index := uval % uint64(num_things)
	return things[index]
}
