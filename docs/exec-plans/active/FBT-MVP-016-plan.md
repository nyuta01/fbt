# FBT-MVP-016 Complete MVP conformance, packaging, and release smoke

## Observation

The MVP feature set is implemented behind Go tests and local smokes, but
security conformance and release-binary checks are not yet first-class
verification targets.

## Decision

Add deterministic gates:

- conformance smoke for policy denial, upstream/confidence blocking, build,
  eval, and docs-site build
- dist check that builds a local release binary and exercises core commands
- wire both checks into `make verify`
- update docs and task state to reflect MVP completion

## Permanent Fix

Added `tests/conformance/run.sh`, `scripts/dist-check.sh`, `make conformance`,
`make dist-check`, and wired both checks into `make verify`.

## Next Check

Run:

```sh
make verify
```

Result on 2026-05-28: pass.
