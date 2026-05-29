# FBT-LOCK-001 Design Optional Runner And Adapter Lockfile Semantics

## Observation

Official adapters can be installed and released, but a project cannot yet pin
the exact runner or adapter source, version, checksum, and protocol
compatibility in a local contract without risking scope creep toward package
management.

## Decision

Define `fbt.lock.json` as an optional validator-only contract. It can pin
runner/adapter identity and integrity metadata, but installation, update,
resolution, and registry access remain outside core.

## Permanent Fix

Added the lockfile spec, JSON schema, discovery/config/schema/adapter docs, and
`runner-lockfile-spec-check` so the validator-only boundary is mechanically
guarded by `make verify`.

## Next Check

`make verify` must pass; future implementation should add doctor/build behavior
against this contract without adding downloads or dependency resolution to core.
