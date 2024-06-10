//go:build !no_antithesis_sdk && windows

package internal

import (
	"errors"
)

func openSharedLib(_ string) (interface{}, error) {
	return nil, errors.New(useLocalHandler)
}
