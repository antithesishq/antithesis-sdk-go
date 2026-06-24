module example.com/multimodule-test

go 1.24.7

toolchain go1.24.8

require (
	example.com/multimodule-test/sub v0.0.0
	github.com/antithesishq/antithesis-sdk-go v0.0.0
)

replace example.com/multimodule-test/sub => ./sub

replace github.com/antithesishq/antithesis-sdk-go => ../../../..
