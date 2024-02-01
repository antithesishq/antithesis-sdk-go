// Package random provides callers with input from the [Antithesis testing platform].
//
// When run inside of Antithesis, callers that modify program behavior
// with the results of these calls will see greater test effeciency and
// faster exporation of their code. Callers running outside of the Antithesis
// environment will be provided values derived from [crypto/rand].
//
// [Antithesis testing platform]: https://antithesis.com
package random

import (
	"github.com/antithesishq/antithesis-sdk-go/internal"
)

// Returns a uint64 value chosen by the Antithesis environment. Test harnesses
// that modify behavior based on such choices will see greater test coverage
// and faster exploration of their code. Ideally this and other functions from
// this package are called repeatedly during the course of a test, not just
// at startup.
func GetRandom() uint64 {
	return internal.Get_random()
}

// Callers allow Antithesis to select one of a set of choices. Test harnesses
// that modify behavior based on such choices will see greater test coverage
// and faster exploration of their code. Ideally this and other functions from
// this package are called repeatedly during the course of a test, not just
// at startup.
func RandomChoice(things []any) any {
	num_things := len(things)
	if num_things < 1 {
		return nil
	}

	uval := GetRandom()
	index := uval % uint64(num_things)
	return things[index]
}
