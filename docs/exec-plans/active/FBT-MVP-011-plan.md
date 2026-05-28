# FBT-MVP-011 Implement policy and security enforcement

## Observation

The build lifecycle validates output candidates stay under the work directory,
but policy checks are still shallow. Core needs deterministic guards for logical
artifact containment, declared write scope, output-size limits, timeout limits,
redaction helpers, and failed-run state safety.

## Decision

Implement a policy/security baseline:

- validate logical output paths stay under `artifact_path`
- validate policy write scope before official commit
- enforce `limits.max_output_bytes` from descriptors
- derive per-transform timeout from policy limits
- provide secret redaction helper for diagnostics/run records
- add tests that denied outputs do not update official current state

## Permanent Fix

Added policy tests for artifact path containment, write scope, output-size
limits, timeout extraction, redaction, and a build test proving denied policy
outputs do not update official artifact state.

## Next Check

Run:

```sh
make verify
```

Result on 2026-05-28: pass.
