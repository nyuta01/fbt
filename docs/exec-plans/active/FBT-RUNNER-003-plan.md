# FBT-RUNNER-003 Enforce runner capability negotiation in doctor and build

## Observation

Runner `initialize` responses were checked for protocol-level success, but core
did not reject runners whose negotiated capabilities could not satisfy the
selected transform type, output artifact types, or output-candidate contract.

## Decision

Treat `initialize` capabilities as authoritative for the current invocation.
Build must stop before `fbt/runTransform` when the runner is incompatible, and
doctor/runner validate must report the same mismatch as a diagnostic.

## Permanent Fix

Added capability validation for protocol version, transform types, artifact
types, and output-candidate support. Wired it into build, `fbt doctor`, and
`fbt runner doctor/validate`, with Go tests and a conformance scenario for an
incompatible runner.

## Next Check

Run:

```sh
make verify
```
