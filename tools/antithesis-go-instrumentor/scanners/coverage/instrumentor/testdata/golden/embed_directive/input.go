package sample

import _ "embed"

// The //go:embed directive must survive comment-trimming and stay attached
// to its var, or the instrumented file won't compile.

//go:embed input.go
var data string

func Data() string {
	return data
}
