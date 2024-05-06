//go:build !no_antithesis_sdk

// Package random requests both structured and unstructured randomness from the Antithesis environment. is part of the [Antithesis Go SDK], which enables Go applications to integrate with the [Antithesis platform].
//
// These functions should not be used to seed a conventional PRNG, and should not have their return values stored and used to make a decision at a later time. Doing either of these things makes it much harder for the Antithesis platform to control the history of your program's execution, and also makes it harder for Antithesis to learn which inputs provided at which times are most fruitful. Instead, you should call a function from the random package every time your program or [workload] needs to make a decision, at the moment that you need to make the decision.
//
// These functions are also safe to call outside the Antithesis environment, where they will fall back on values from [crypto/rand].
// [Antithesis Go SDK]: https://antithesis.com/docs/using_antithesis/sdk/go_sdk.html
// [Antithesis platform]: https://antithesis.com
// [workload]: https://antithesis.com/docs/getting_started/workload.html
package random

import (
	"github.com/antithesishq/antithesis-sdk-go/internal"
)

// GetRandom returns a uint64 value chosen by Antithesis. You should not store this value or use it to seed a PRNG, but should use it immediately.
func GetRandom() uint64 {
	return internal.Get_random()
}
