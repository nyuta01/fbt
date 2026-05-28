# FBT-H-004 Define security model and conformance scenarios

## Observation

The design described policy, scoped writes, confidence gates, and secret
redaction, but those ideas were not yet expressed as deterministic acceptance
scenarios. Without conformance scenarios, implementation could drift toward
happy-path execution.

## Decision

Pin the MVP security baseline:

- core owns official commit, state, descriptors, artifact versions, and evals
- runner stdout, stderr, output paths, and generated content are untrusted
- output candidates must stay under invocation work directories
- logical artifact paths must stay under `artifact_path`
- failed, cancelled, interrupted, or denied runs cannot update official pointers
- confidence is bound to artifact versions
- conformance uses fake runners and requires no external services

## Permanent Fix

Added `docs/security-and-conformance-spec.md`, linked it from core and runner
protocol specs, and defined the initial conformance scenario table.

## Next Check

Run:

```sh
make verify
```

When product code begins, add a `make conformance` target under `make verify`
using fake runners for path escapes, policy denial, state safety, confidence
blocking, dirty-state, and redaction scenarios.
