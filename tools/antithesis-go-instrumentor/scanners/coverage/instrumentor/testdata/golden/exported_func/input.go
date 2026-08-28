package sample

// A //export directive means cgo may take the address of this function;
// AST rewriting could break it, so the whole file is skipped (empty output).

//export MyExport
func MyExport() {
	println("hi")
}
