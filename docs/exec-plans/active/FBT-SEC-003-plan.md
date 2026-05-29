# FBT-SEC-003 Plan

## Task

Validate production sandbox and credential profiles for the daily support
operation.

## Observation

The repository documented OS sandbox profiles, but the production daily loop
did not record which external profile was used or scan handoff files for
credential leaks.

## Decision

Keep OS sandboxing outside fbt core. Add a daily ops security-profile script
that records the selected external profile and scans run/publish handoff files
for configured secret values.

## Permanent Fix

`daily-ops-smoke` now sets a synthetic secret marker, runs the security profile
handoff, verifies the pass result, and asserts the marker is absent from the run
and publish handoff files.

## Next Check

- `make daily-ops-smoke`
- `make security-profiles-check`
- `make verify`
