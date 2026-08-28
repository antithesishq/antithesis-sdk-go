//go:build !no_antithesis_sdk && linux && amd64 && !cgo

package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNoCgoRunsInDegradedMode(t *testing.T) {
	if !sdkRunningInDegradedMode {
		t.Fatal("expected sdkRunningInDegradedMode to be true in the no-cgo build")
	}
}

func TestNoCgoInitReturnsNilWithoutOutputDir(t *testing.T) {
	os.Unsetenv(outputDirEnvVar)
	if h := init_in_antithesis(); h != nil {
		t.Fatalf("expected nil handler when %s is unset, got %T", outputDirEnvVar, h)
	}

	os.Setenv(outputDirEnvVar, "")
	defer os.Unsetenv(outputDirEnvVar)
	if h := init_in_antithesis(); h != nil {
		t.Fatalf("expected nil handler when %s is empty, got %T", outputDirEnvVar, h)
	}
}

func TestNoCgoInitWritesToSdkJsonl(t *testing.T) {
	dir := t.TempDir()
	os.Setenv(outputDirEnvVar, dir)
	defer os.Unsetenv(outputDirEnvVar)

	h := init_in_antithesis()
	if h == nil {
		t.Fatalf("expected a handler when %s is set", outputDirEnvVar)
	}
	if _, ok := h.(*inAntithesisWithoutCgoHandler); !ok {
		t.Fatalf("expected *inAntithesisWithoutCgoHandler, got %T", h)
	}

	h.output(`{"antithesis_sdk":"first"}`)
	h.(*inAntithesisWithoutCgoHandler).outputFile.Close()

	path := filepath.Join(dir, fallbackOutputFilename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
	if got := string(data); got != `{"antithesis_sdk":"first"}`+"\n" {
		t.Fatalf("unexpected file contents: %q", got)
	}
}
