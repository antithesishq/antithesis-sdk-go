// Command runtime_caller confirms the instrumentor leaves runtime.Caller alone
// for files that use it. Because no leading //line directive is emitted for such
// files, runtime.Caller reports the real (absolute) compiled path rather than
// the instrumentor's relative source path. caller() asserts the path is absolute
// and panics otherwise, so a clean exit proves the //line was correctly skipped;
// a regression (directive re-added) makes runtime.Caller return a relative path
// and the program panics, failing the test. Instrumentation is otherwise
// unaffected, so edges still fire normally. Args: one integer.
package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

var sink string

func caller() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok || !filepath.IsAbs(file) {
		panic("runtime.Caller returned a non-absolute path: " + file)
	}
	return file
}

func classify(n int) string {
	if n < 0 {
		return "neg"
	}
	return "nonneg"
}

func main() {
	sink = caller()
	n, _ := strconv.Atoi(os.Args[1])
	sink = classify(n)
}
