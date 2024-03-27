//go:build !antithesis_sdk

package random

import (
	"math/rand"
)

func GetRandom() uint64 {
	return rand.Uint64()
}
