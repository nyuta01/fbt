# FBT-CI-001 Plan

## Task

Define the authoritative CI builder and reproducibility handoff.

## Observation

The daily wrapper created production evidence, but the team operation still
needed one clear rule: local runs are for iteration, CI is the official builder
that pins versions and archives the run evidence.

## Decision

Update the copyable GitHub Actions workflow to pin fbt, validate the source
window, run the daily wrapper with a CI security profile, and upload run,
archive, and publish handoff evidence. Document the same rule in the daily ops
guide and release docs.

## Permanent Fix

`make verify` now includes `ci-authority-check`, which scans the workflow and
docs for the authoritative-builder contract, version pinning, archive upload,
and run-bundle handoff.

## Next Check

- `make ci-authority-check`
- `make verify`
