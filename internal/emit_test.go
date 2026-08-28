//go:build !no_antithesis_sdk

package internal

import (
	"encoding/json"
	"os"
	"testing"
)

func TestLocalHandlerFileOutput(t *testing.T) {
	path := t.TempDir() + string(os.PathSeparator) + "antithesis-test.log"
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
