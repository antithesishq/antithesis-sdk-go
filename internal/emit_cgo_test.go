//go:build !no_antithesis_sdk && linux && amd64 && cgo

package internal

import (
	"os"
	"testing"
)

func TestVoidstarHandlerErr1(t *testing.T) {
	_, err := openSharedLib("path-not-exists")
	if err == nil {
		panic("Should failed to load library")
	}
}

func TestVoidstarHandlerErr2(t *testing.T) {
	_, err := openSharedLib(os.Args[0])
	if err == nil {
		panic("Should failed to load library")
	}
}

func TestVoidstarHandlerErr3(t *testing.T) {
	_, err := openSharedLib("libc.so.6")
	if err == nil {
		panic("Should failed to load library")
	}
}
