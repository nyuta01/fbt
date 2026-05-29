# FBT-LOCK-002 Implement Validator-Only Runner Lockfile Diagnostics

## Observation

The optional `fbt.lock.json` contract was specified, but users could not rely
on `doctor` or `build` to detect runner command, protocol, checksum, or
capability drift from that contract.

## Decision

Implement lockfile reading as validation only. `doctor` explains drift, `build`
fails before `fbt/runTransform` for selected incompatible runners, and no
command downloads, installs, updates, or resolves packages.

## Permanent Fix

The lockfile is wired into runner identity, diagnostics, and preflight
validation while preserving the package-manager boundary. Valid matching lock
entries participate in runner fingerprints, doctor reports missing/unused/
mismatched entries, and failed selected builds record `runner_lock_incompatible`
receipts without committing artifacts.

## Next Check

Done. Unit, CLI, and conformance coverage exercise valid lockfiles, command /
protocol / checksum / capability mismatches, missing and unused entries, dirty
state participation, malformed lockfiles, and validator-only behavior. `make
verify` passed.
