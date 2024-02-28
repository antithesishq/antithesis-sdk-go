#! /bin/sh
set -e
go fmt -x github.com/antithesishq/antithesis-sdk-go/assert
go fmt -x github.com/antithesishq/antithesis-sdk-go/instrumentation
go fmt -x github.com/antithesishq/antithesis-sdk-go/internal
go fmt -x github.com/antithesishq/antithesis-sdk-go/lifecycle
go fmt -x github.com/antithesishq/antithesis-sdk-go/random

go fmt -x github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor
go fmt -x github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/cmd
go fmt -x github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common
go fmt -x github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/instrumentor
go fmt -x github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/assertions

go build assert/*.go
go build lifecycle/*.go
go build internal/*.go
go build random/*.go
go build instrumentation/*.go

go install tools/antithesis-go-instrumentor/*.go
