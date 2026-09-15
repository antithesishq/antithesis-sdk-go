
# Antithesis Go SDK

This library provides methods for Go programs to configure the [Antithesis](https://antithesis.com) platform. Functionality is grouped into the packages [`assert`](https://antithesis.com/docs/generated/sdk/golang/assert/) for defining new test properties, [`random`](https://antithesis.com/docs/generated/sdk/golang/random/) for Antithesis input, and [`lifecycle`](https://antithesis.com/docs/generated/sdk/golang/lifecycle/) for controlling the Antithesis simulation.

For general usage guidance see the [Antithesis Go SDK Documentation](https://antithesis.com/docs/using_antithesis/sdk/go/)

## Compiling the SDK in

On this branch the SDK is compiled out by default. Building without
`-tags=enable_antithesis_sdk` leaves every assertion, lifecycle and coverage
entry point as an empty function, and `random.GetRandom` backed by
`crypto/rand`.

Those empty functions are still ordinary Go calls, so their arguments are
evaluated at the call site. Work done only to produce a condition or a
`details` map is therefore still paid for. Guard it with `assert.Enabled`,
which is a constant, so the compiler discards the whole block:

```go
if assert.Enabled {
    st := svc.SnapshotState()
    assert.Always(st.Valid(), "state is valid", map[string]any{"state": st})
}
```

Assertions inside the guard are still discovered by the instrumentor and still
registered in the generated assertion catalog, so no test property is lost.

Write the guard in that positive form, with no `else`. It is the only shape the
coverage instrumentor recognizes, and recognizing it lets the instrumentor skip
the "not taken" coverage edge it would otherwise synthesize — an edge that
could never be reached, since the condition is a constant.

Note that `assert.Enabled` reports only whether the SDK was *compiled in*. It
does not report whether the program is running under Antithesis; an enabled
build elsewhere still reports true and simply falls back to a local or silent
output handler.

One existing limitation is worth knowing about, independent of the above: the
instrumentor recognizes assertion calls through a package qualifier, so
assertions reached through a dot-imported `assert` package are not cataloged.
Plain and aliased imports both work.
