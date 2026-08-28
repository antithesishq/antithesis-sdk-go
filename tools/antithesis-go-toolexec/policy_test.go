package main

import (
	"strings"
	"testing"

	qt "github.com/go-quicktest/qt"
)

// The argument lists below are real `go build -toolexec` compile invocations,
// captured from go1.26.6 with $WORK abbreviated.
func TestSourceFiles(t *testing.T) {
	tests := []struct {
		name  string
		args  string
		files []string
	}{{
		name:  "single file",
		args:  "-o $W/b001/_pkg_.a -trimpath $W/b001=> -p main -lang=go1.24 -complete -buildid abc/abc -goversion go1.26.6 -c=24 -nolocalimports -importcfg $W/b001/importcfg -pack ./main.go",
		files: []string{"./main.go"},
	}, {
		name:  "several files",
		args:  "-o $W/b006/_pkg_.a -p internal/goarch -lang=go1.26 -std -complete -importcfg $W/b006/importcfg -pack /goroot/src/internal/goarch/goarch.go /goroot/src/internal/goarch/goarch_amd64.go /goroot/src/internal/goarch/zgoarch_amd64.go",
		files: []string{"/goroot/src/internal/goarch/goarch.go", "/goroot/src/internal/goarch/goarch_amd64.go", "/goroot/src/internal/goarch/zgoarch_amd64.go"},
	}, {
		name:  "embedcfg does not look like a source file",
		args:  "-o $W/b001/_pkg_.a -p main -lang=go1.24 -complete -importcfg $W/b001/importcfg -embedcfg $W/b001/embedcfg -pack /src/main.go",
		files: []string{"/src/main.go"},
	}, {
		name:  "no source files at all",
		args:  "-V=full",
		files: nil,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := strings.Fields(test.args)
			var got []string
			for _, file := range sourceFiles(args) {
				qt.Assert(t, qt.Equals(args[file.Index], file.Path))
				got = append(got, file.Path)
			}
			qt.Assert(t, qt.DeepEquals(got, test.files))
		})
	}
}

func TestSkipFile(t *testing.T) {
	excluded := map[string]bool{"/src/vendored": true, "/src/one.go": true}
	cfg := &Config{Exclusions: excluded}

	qt.Assert(t, qt.IsTrue(skipFile(cfg, "/src/one.go")))
	qt.Assert(t, qt.IsTrue(skipFile(cfg, "/src/vendored/deep/two.go")))
	qt.Assert(t, qt.IsTrue(skipFile(cfg, "/src/api.pb.go")))
	qt.Assert(t, qt.IsFalse(skipFile(cfg, "/src/three.go")))
	// A directory sharing a prefix with an exclusion is not itself excluded.
	qt.Assert(t, qt.IsFalse(skipFile(cfg, "/src/vendored_other/four.go")))

	qt.Assert(t, qt.IsFalse(skipFile(cfg, "/src/five_test.go")))
	cfg.SkipTestFiles = true
	qt.Assert(t, qt.IsTrue(skipFile(cfg, "/src/five_test.go")))
}

func TestInScope(t *testing.T) {
	cfg := &Config{Instrument: []string{"example.com/app"}}

	qt.Assert(t, qt.IsTrue(inScope(cfg, "example.com/app", "/src/a.go")))
	qt.Assert(t, qt.IsTrue(inScope(cfg, "example.com/app/worker", "/src/a.go")))
	// A prefix match must respect path boundaries.
	qt.Assert(t, qt.IsFalse(inScope(cfg, "example.com/application", "/src/a.go")))
	qt.Assert(t, qt.IsFalse(inScope(cfg, "example.com/other", "/src/a.go")))
}

func TestIsSDKPackage(t *testing.T) {
	qt.Assert(t, qt.IsTrue(isSDKPackage("github.com/antithesishq/antithesis-sdk-go")))
	qt.Assert(t, qt.IsTrue(isSDKPackage("github.com/antithesishq/antithesis-sdk-go/instrumentation")))
	qt.Assert(t, qt.IsFalse(isSDKPackage("github.com/antithesishq/antithesis-sdk-goodies")))
	qt.Assert(t, qt.IsFalse(isSDKPackage("example.com/app")))
}

func TestIsVersionProbe(t *testing.T) {
	qt.Assert(t, qt.IsTrue(isVersionProbe([]string{"-V=full"})))
	qt.Assert(t, qt.IsFalse(isVersionProbe(nil)))
	// The C compiler is probed differently, and its output is hashed rather than
	// parsed; we must leave that invocation alone.
	qt.Assert(t, qt.IsFalse(isVersionProbe([]string{"-###", "-x", "c", "-c", "-"})))
	qt.Assert(t, qt.IsFalse(isVersionProbe([]string{"-o", "x.a", "-pack", "a.go"})))
}

func TestWithoutToolexec(t *testing.T) {
	qt.Assert(t, qt.Equals(withoutToolexec("-toolexec=/bin/wrap -mod=mod"), "-mod=mod"))
	qt.Assert(t, qt.Equals(withoutToolexec("-mod=mod"), "-mod=mod"))
	qt.Assert(t, qt.Equals(withoutToolexec(""), ""))
}

func TestEscapeImportPath(t *testing.T) {
	for _, importPath := range []string{"app", "example.com/app/worker", "a/b/c"} {
		qt.Assert(t, qt.Equals(unescapeImportPath(escapeImportPath(importPath)), importPath))
	}
}
