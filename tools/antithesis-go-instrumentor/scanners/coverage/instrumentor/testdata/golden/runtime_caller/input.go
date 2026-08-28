package sample

import "runtime"

// Where uses runtime.Caller, so the instrumentor must NOT emit its leading
// //line directive: a static //line path would override the path runtime.Caller
// reports at runtime (golang/go#26207). The file is still instrumented (Notify
// calls inserted); it just carries no //line, so expected.golden has none. This
// matches the AST-rewriting backend's behavior for such files.
func Where() string {
	_, file, _, _ := runtime.Caller(0)
	return file
}
