//go:build !no_antithesis_sdk

package internal

import (
	"os"
	"testing"
)

func TestLocalOutputReusesBufferCapacity(t *testing.T) {
	file, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	h := localHandler{outputFile: file}
	// A large message prevents stack allocation from hiding a string-copy regression.
	payload := make([]byte, 4096, 4097)
	if allocations := testing.AllocsPerRun(100, func() { h.output(payload) }); allocations != 0 {
		t.Fatalf("local output allocated %g times with spare buffer capacity", allocations)
	}
}
