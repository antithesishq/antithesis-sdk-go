package main

import (
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sync"
)

// instrumentorIdentity identifies this build of the wrapper, for the tool ID
// reported through -V=full.
var instrumentorIdentity = sync.OnceValues(func() (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locating this executable in order to identify it: %w", err)
	}

	// A binary installed at a version (go install ...@v1.2.3) carries it, and it
	// is stable across machines, so prefer it when it is there.
	if info, err := buildinfo.ReadFile(self); err == nil {
		if version := info.Main.Version; version != "" && version != "(devel)" {
			return "version=" + version, nil
		}
	}

	// If an artifact without a version is being used, use the SHA256 of the executable instead
	file, err := os.Open(self)
	if err != nil {
		return "", fmt.Errorf("reading this executable in order to identify it: %w", err)
	}
	defer file.Close()

	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", fmt.Errorf("hashing this executable in order to identify it: %w", err)
	}
	return "content=" + hex.EncodeToString(digest.Sum(nil))[:16], nil
})
