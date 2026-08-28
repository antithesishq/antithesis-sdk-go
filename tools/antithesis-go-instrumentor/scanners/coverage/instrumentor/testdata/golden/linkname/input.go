package sample

import _ "unsafe"

// A go:linkname directive ties this file to another language/symbol; the
// instrumentor skips the whole file (empty output).

//go:linkname localName runtime.nanotime
func localName() int64
