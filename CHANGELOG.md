# Changelog

## 0.9.0 - 2026-11-10

Numeric guidance now compares `float64(left) - float64(right)`, the difference
the platform itself computes from the operands it receives, instead of an exact
integer difference.

The instrumentor's symbol table gains a `class` column holding a method's receiver type, and names function literals the way the compiler does (`Outer.func1`, nested `Outer.func1.1`, `glob..func1` at package level) instead of the enclosing declaration's name or nothing at all, so same-named methods and closures in one file no longer merge.

Greatly increased the efficiency and reduced allocations done by all assertion types.

Nil details are omitted from assertions and setup notifications; explicit empty
maps remain empty objects. 

Assertion declarations always omit details.

Lifecycle calls wrap non-object JSON values as `{"value": v}`, and nil event payloads become
`{}`. 

Details serialization failures emit an `antithesis_error` with the call's
context and still emit the assertion or event without details. 

Lifecycle details now share assertions' string rendering of errors in maps and slices, except for errors with custom JSON marshalers. 

Normalization copies containers only when an
error needs this rendering, and rejects cyclic details without overflowing the stack.

Non-finite floats are replaced by the strings `"NaN"`, `"Infinity"` or `"-Infinity"`.

Breaking: `random.RandomChoice` now returns `(T, bool)`, with `false` for an empty slice. It previously returned the zero value, which was indistinguishable from a real element.

## 0.8.0 - 2026-08-28

Adds a new -source_editing flag to the `antithesis-go-instrumentor` tool that
is designed to maintain compiler directives such as `//go:embed` with higher
fidelity to the original source.

Significant performance improvements for
assertions.

Clean up some inconsistencies between the initialization messages emitted by
the Go SDK vs. others.

Upgrade to instrumentation ABI v2 -- Go now supports coverage leases.

When used in local debug mode, the output file (`ANTITHESIS_SDK_LOCAL_OUTPUT`)
will no longer be truncated at initialization.

`assert.AssertRaw` takes a `column` parameter after `line`, like the raw assertion entry points of the other SDKs. The high-level assertion functions and the instrumentor's generated registrations still record column 0.

## 0.7.2 - 2026-05-13

Fix `PathFromBaseDirectory` mispathing multi-level submodules. Its
`filepath.Match` pattern was `baseDir/*`, which does not cross path
separators — so any customer submodule nested two or more levels deep
had its modified `go.mod` written to `customer/<input-abspath>/...`
instead of `customer/<rel-path>/...`. Now uses `filepath.Rel`. Affected
any non-trivial Go monorepo (etcd, k8s, etc.) since the instrumented
output of deep submodules was non-functional. (ENG-3940)

## 0.7.1 - 2026-05-13

Fix the instrumentor leaking the host's Go version into the notifier
module's `go.mod`, which caused `go mod tidy` to bump every customer
module's `go` directive (and drop `toolchain`) just because it now
required the notifier. The notifier's `go` directive is now pinned to
the minimum across the customer modules the instrumentor touches, and
`toolchain` is omitted.

## 0.7.0 - 2026-03-20

Fix assertion cataloging for Go modules that produce multiple binaries.

## 0.6.0 - 2026-02-12

Implementing a `rand.Source` for the Antithesis platform's randomness to integrate with `rand.Rand`.

## 0.5.0 - 2025-08-28

Support for `error` values passed into assertion `details` fields.

## 0.4.4 - 2025-07-03

Fix assertion scanning so it actually runs in `-assert_only` mode.

## 0.4.3 - 2024-12-16

Fix file tree copying in the instrumentor.

Improve default behavior for instrumentation package version selection.

Instrument all `.go` files by default.

## 0.4.2 - 2024-10-30

Cleanup and bug fixes related to the notifier module.

## 0.4.1 - 2024-09-20

Add the notifier module to all submodules that are instrumented.

## 0.4.0 - 2024-07-10

Adding guidance-based assertions. These are both assertions and guidance for the fuzzer to explore your program more effectively.

Improvements to `RandomChoice`

Updated `cp` command to support alpine and other distros.

## 0.3.8 - 2024-05-17

Preventing an internal race condition.

Improvements to default message.

## 0.3.7 - 2024-05-17

Supporting a default message in assertions if none is provided.

## 0.3.6 - 2024-05-08

Improvements to assertion cataloging and to documentation.

## 0.3.5 - 2024-05-02

Fixing a bug where instrumentor `cp` didn't work on MacOS.
