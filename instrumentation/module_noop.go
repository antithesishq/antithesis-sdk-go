//go:build !enable_antithesis_sdk

package instrumentation

type Module struct{}

func RegisterModule(symbolTable string, edgeCount int) *Module { return &Module{} }

func (m *Module) Notify(edge int) {}
