//go:build !antithesis_sdk

package instrumentation

func InitializeModule(symbolTable string, edgeCount int) uint64 {
	return 0
}

func Notify(edge int) {}
