package assert

// Rich Assertion numeric guideposts can use any of these
type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint8 | ~uint16 | ~uint32 | ~float32 | ~float64 | ~uint64 | ~uint | ~uintptr
}

// Internally, numeric guidepost Operands only use these
type Operand interface {
	int32 | int64 | uint64 | float64
}

type NumType interface {
	uint64 | float64
}

type Pair struct {
	First  string `json:"first"`
	Second bool   `json:"second"`
}
