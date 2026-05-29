# FBT-LOCK-002 Implement Validator-Only Runner Lockfile Diagnostics

## Observation

The optional `fbt.lock.json` contract is specified, but users cannot yet rely on
`doctor` or `build` to detect runner command, protocol, checksum, or capability
drift from that contract.

## Decision

Implement lockfile reading as validation only. `doctor` should explain drift,
`build` should fail before runner execution for selected incompatible runners,
and no command should download, install, update, or resolve packages.

## Permanent Fix

Wire the lockfile into runner identity, diagnostics, and preflight validation
while preserving the package-manager boundary.

## Next Check

Add conformance cases for valid lockfiles, mismatches, unused entries, missing
entries, and no-network/no-install behavior, then run `make verify`.
