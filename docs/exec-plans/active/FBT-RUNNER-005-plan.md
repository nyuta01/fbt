# FBT-RUNNER-005 Add runner authoring kit and conformance fixtures

## Observation

External runner authors could infer the stdio protocol from specs and Go tests,
but there was no small black-box fixture that validated a runner command the
same way a third-party author would use it.

## Decision

Add a language-neutral conformance harness outside core. The harness should
spawn any runner command, exercise `initialize` and `fbt/runTransform`, validate
capabilities and output-candidate containment, and remain deterministic enough
to run in `make verify` against the source fake runner.

## Permanent Fix

Added `tests/runner-conformance/run.py`, sample JSON fixtures, a shell wrapper,
`make runner-conformance`, and a runner authoring guide. `make verify` now runs
the strict conformance fixture by default against `runners/fake`.

## Next Check

Run:

```sh
make verify
```
