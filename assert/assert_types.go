package assert

type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint8 | ~uint16 | ~uint32 | ~float32 | ~float64 // | ~uint64 | ~uint
}

type Pair struct {
	First  string `json:"first"`
	Second bool   `json:"second"`
}
