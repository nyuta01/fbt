# FBT-RUNNER-025 Expose runner stderr and exit diagnostics in protocol failures

## Observation

The protocol client starts the runner with stdout and stdin pipes only. If a
runner exits before initialization or writes diagnostic details to stderr, the
core error path can collapse to a generic protocol or EOF error.

## Decision

Capture bounded stderr and process exit status in the protocol client, then
surface safe diagnostics through failed build receipts and human CLI hints.
Secret redaction rules must still apply to configured runner environment
values.

## Permanent Fix

Add tests for a runner that exits before `initialize` and writes stderr. Verify
that `build` records a safe error kind/message and that human output gives an
actionable hint without leaking secrets.

## Next Check

Run protocol, build, and CLI tests, then `make verify`.
