# FBT-UX-005 Add top-level project doctor command

## Observation

Users had to combine `parse`, `state status`, and `runner doctor` manually to
know whether a project was ready to build. Runner executable checks existed,
but the top-level readiness flow did not verify protocol initialization.

## Decision

Add `fbt doctor` as a local-first readiness check. It should parse the project,
check state lock and writability, check runner discovery/executability, and run
runner protocol `initialize` without adding a daemon or background service.

## Permanent Fix

Implemented `fbt doctor` with text and JSON output, exit code `6` for runner or
dependency readiness failures, CLI tests for passing and failing projects, CLI
smoke coverage, and user-facing docs.

## Next Check

Run:

```sh
make verify
```
