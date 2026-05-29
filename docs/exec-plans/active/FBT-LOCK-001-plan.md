# FBT-LOCK-001 Design Optional Runner And Adapter Lockfile Semantics

## Observation

Official adapters can be installed and released, but a project cannot yet pin
the exact runner or adapter source, version, checksum, and protocol
compatibility in a local contract.

## Decision

Design an optional `fbt.lock.json` model for runner and adapter reproducibility.
Keep installation and dependency resolution outside core; fbt should only read,
validate, and explain lockfile expectations when present.

## Permanent Fix

Specify the lockfile shape, the dirty/doctor behavior when runner identity
drifts, and conformance coverage that proves core remains a validator rather
than a package manager.

## Next Check

Update specs/docs, add focused validation behavior if accepted, and run
`make verify`.

