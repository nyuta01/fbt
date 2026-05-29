# FBT-RUNNER-025 Expose runner stderr and exit diagnostics in protocol failures

## Observation

The protocol client starts the runner with stdout and stdin pipes only. If a
runner exits before initialization or writes diagnostic details to stderr, the
core error path can collapse to a generic protocol or EOF error.

That makes real external adapter failures hard to diagnose because the useful
provider, CLI, or credential setup message is often on stderr.

## Decision

Capture bounded stderr and process exit status in the protocol client, then
surface safe diagnostics through failed build receipts and human CLI hints.
Secret redaction rules must still apply to configured runner environment
values.

The protocol client now captures up to 32 KiB of runner stderr and includes
exit status when stdout closes before the expected response or the request pipe
breaks. Values passed through configured runner `env` names are redacted before
the error reaches build receipts or CLI output.

## Permanent Fix

Add tests for a runner that exits before `initialize` and writes stderr. Verify
that `build` records a safe error kind/message and that human output gives an
actionable hint without leaking secrets.

## Next Check

Done. Protocol, build, and CLI tests cover redacted stderr/exit diagnostics,
failed build receipts, and the CLI hint. `make verify` passes.
