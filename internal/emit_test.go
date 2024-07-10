//go:build enable_antithesis_sdk

package internal

import (
	"encoding/json"
	"os"
	"testing"
)

var test_result bool

func TestLocalHandlerFileOutput(t *testing.T) {
	path := os.TempDir() + string(os.PathSeparator) + "antithesis-test.log"
	os.Setenv(localOutputEnvVar, path)
	defer os.Unsetenv(localOutputEnvVar)
	handler = openLocalHandler()
	Json_data(map[string]string{
		"test": "output",
	})
	handler.(*localHandler).outputFile.Close()
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var result map[string]string
	if err = json.Unmarshal(data, &result); err != nil {
		panic(err)
	}
	if result["test"] != "output" {
		panic("JSON does not roundtrip")
	}
}

func TestLocalHandlerNop(t *testing.T) {
	os.Setenv(localOutputEnvVar, "")
	defer os.Unsetenv(localOutputEnvVar)
	handler = openLocalHandler()
	Json_data(map[string]string{
		"test": "output",
	})
	h, valid := handler.(*localHandler)
	if !valid {
		panic("Not using the local handler")
	}
	if h.outputFile != nil {
		panic("Should not be outputting to file")
	}
}

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
