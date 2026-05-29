# FBT-RUNNER-014 Adopt monorepo adapter module strategy

## Observation

Official runner adapters should be maintained as first-class packages, but
starting them in separate repositories would add repository and release
management overhead before the package model is proven.

## Decision

Use one repository for now, with strict module boundaries:

- root `go.mod` remains fbt core only
- `sdk/go` becomes the provider-free runner SDK module
- `adapters/<name>` becomes the official adapter module location
- `go.work` is allowed for local development convenience
- provider SDKs and agent runtimes stay inside adapter modules, never core
- future language SDKs such as `sdk/python` or `sdk/typescript` remain possible

## Permanent Fix

Updated the official adapter design report and runner adapter packaging docs to
state the monorepo nested-module strategy, the `sdk/` and `adapters/` target
layout, and the rule that the protocol spec plus conformance suite remain the
source of truth across languages.

## Next Check

Done. `make verify` passes. Next implementation should create `sdk/go`, then
promote the command and OpenAI adapters under `adapters/`.
