# Changelog

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

