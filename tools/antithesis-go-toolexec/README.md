# antithesis-go-toolexec

Adds Antithesis coverage instrumentation and assertion catalogs from inside
`go build`, as a
[`-toolexec`](https://pkg.go.dev/cmd/go#hdr-Compile_packages_and_dependencies)
wrapper, instead of by copying and rewriting a source tree.

## Using it

```sh
go build -toolexec=antithesis-go-toolexec -o /app ./cmd/app
```

or, so that no build command has to change:

```dockerfile
ENV GOFLAGS=-toolexec=antithesis-go-toolexec
RUN go build -o /app ./cmd/app
RUN go test ./...                      # instrumented too
```

Assertion catalogs are produced too, one per instrumented package, so no separate
`antithesis-go-instrumentor` pass is needed.

Coverage and cataloguing are independent. For catalogs alone, turn coverage off:

```sh
ANTITHESIS_SKIP_COVERAGE=1 go build -toolexec=antithesis-go-toolexec ./...
```

Or to produce symbol tables without assertion catalogs, turn cataloguing off:

```sh
ANTITHESIS_SKIP_CATALOG=1 go build -toolexec=antithesis-go-toolexec ./...
```

### In a multi-stage image build

Symbol tables are written to `/symbols` by default, which is where the image
build harvests them. A multi-stage Dockerfile has to carry them forward:

```dockerfile
FROM golang:1.24 AS builder
ENV GOFLAGS=-toolexec=antithesis-go-toolexec
RUN go build -o /app ./cmd/app

FROM debian:bookworm-slim
COPY --from=builder /app /app
COPY --from=builder /symbols /symbols
```

### Configuration

Configuration is via environment variables:

| Variable | Effect |
| --- | --- |
| `ANTITHESIS_INSTRUMENT` | Comma-separated import path prefixes to instrument. Default: everything whose sources are in the working tree, which is the main module plus any locally replaced module, and excludes downloaded dependencies. |
| `ANTITHESIS_SYMBOLS_DIR` | Where symbol tables are collected. Default `/symbols`. |
| `ANTITHESIS_EXCLUDE` | Exclusions file, in the format `antithesis-go-instrumentor -exclude` accepts. |
| `ANTITHESIS_SYMBOL_PREFIX` | Prefix for symbol table file names, matching `-prefix`. |
| `ANTITHESIS_SDK_MODULE_DIR` | An `antithesis-sdk-go` checkout, used to supply the SDK to a module that does not depend on it. See below. |
| `ANTITHESIS_SKIP_TEST_FILES` | Set to `1` to leave `_test.go` files uninstrumented. |
| `ANTITHESIS_SKIP_CATALOG` | Set to `1` to skip assertion cataloguing. |
| `ANTITHESIS_SKIP_COVERAGE` | Set to `1` to skip coverage instrumentation. Leaving cataloguing on makes this the equivalent of `antithesis-go-instrumentor -assert_only`. |
| `ANTITHESIS_VERBOSE` | `0`–`3`. Logs to stderr during the build. A compile that writes to stderr is not cached by the go command, so this also disables incremental builds — fine for diagnosis, not for a development loop. |
