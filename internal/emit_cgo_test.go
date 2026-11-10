//go:build !no_antithesis_sdk && linux && amd64 && cgo

package internal

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"unsafe"
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

func TestNativeOutputBorrowsBytesThroughFlush(t *testing.T) {
	// Inspect the payload during flush, and return its address only after a valid
	// delivery, so the test catches copies as well as truncation or missing flushes.
	const source = `
#include <stdbool.h>
#include <stdint.h>
#include <stddef.h>
#include <string.h>
static const char *pending;
static size_t pending_size;
static uintptr_t observed;
static unsigned calls;
void fuzz_json_data(const char *data, size_t size) {
 pending = data;
 pending_size = size;
 ++calls;
}
void fuzz_flush(void) {
 const char expected[] = {'a', '\0', 'b'};
 observed = calls == 1 && pending != NULL && pending_size == sizeof(expected) &&
   memcmp(pending, expected, sizeof(expected)) == 0 ? (uintptr_t)pending : 0;
 pending = NULL;
}
uint64_t fuzz_get_random(void) { return observed; }
bool notify_coverage(size_t edge) { return false; }
uint64_t init_coverage_module(size_t edges, const char *symbols) { return 0; }
`
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "output.c")
	libraryPath := filepath.Join(dir, "output.so")
	if err := os.WriteFile(sourcePath, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	compiler := strings.Fields(os.Getenv("CC"))
	if len(compiler) == 0 {
		compiler = []string{"cc"}
	}
	command := exec.Command(compiler[0], append(compiler[1:], "-shared", "-fPIC", sourcePath, "-o", libraryPath)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile native fixture: %v\n%s", err, output)
	}
	h, err := openSharedLib(libraryPath)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte{'a', 0, 'b'}
	h.output(nil)
	h.output([]byte{})
	h.output(payload)
	want := uint64(uintptr(unsafe.Pointer(&payload[0])))
	if got := h.random(); got != want {
		t.Fatalf("native delivery address = %#x, want borrowed buffer %#x", got, want)
	}
	h.output(nil)
	if got := h.random(); got != want {
		t.Fatalf("empty output triggered native delivery or flush: %#x", got)
	}
}
