# FBT-STD-005 Add conformance fixtures for standard exports

## Observation

OpenLineage and OTel exports were covered by Go tests and CLI smoke checks, but
the conformance suite did not verify standard payload shape or redaction across
the end-to-end support workflow.

## Decision

Extend `tests/conformance/run.sh` with generated standard-export fixtures from
the support template. Keep the checks dependency-light by using shell and the
Python standard library, and avoid starting Marquez, an OTel collector, or any
other backend in the default gate.

## Permanent Fix

The conformance run now injects a redaction marker into source and asset files,
builds the support loop, exports OpenLineage and OTel payloads, and
asserts required standard keys, fbt facets, trace/span attributes, runner span
events, and absence of raw source content or the marker secret.

## Next Check

Run:

```sh
make verify
```
