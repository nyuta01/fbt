# FBT-RUNNER-015 Create provider-free Go runner SDK module

## Observation

Official adapters need shared protocol types and small stdio JSON-RPC helpers,
but they must not import fbt core `internal/` packages or bring provider SDKs
into the root module.

## Decision

Created `sdk/go` as a nested Go module and added `go.work` for local
development across the root module and SDK module. The SDK is intentionally
provider-free and contains only protocol types, JSONL stdio JSON-RPC helpers,
output-candidate helpers, and redaction helpers.

## Permanent Fix

`make verify` now includes `sdk-go-test`, so SDK drift is checked with the
normal repository gate while the root `go.mod` stays fbt core only.

## Next Check

Done. `make verify` passes. The next task should promote the command runner
into `adapters/command` using the SDK instead of importing core internals.
