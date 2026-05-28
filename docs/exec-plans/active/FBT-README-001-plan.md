# FBT-README-001 Refresh README using Folio structure

## Observation

The README still read like an internal documentation map even after the MVP
release and docs site publication. Folio's README is a better public entry
point: centered identity, badges, short product definition, surfaces, docs,
quickstart, install, release, and harness sections.

## Decision

Rewrite the fbt README to follow the Folio shape while preserving fbt-specific
boundaries: local-first filesystem artifact control plane, external runners,
standard lineage exports, practical examples, and release integrity.

## Permanent Fix

Updated `README.md` into a public project entry page with release/docs badges,
project layout, surfaces, docs pointers, quickstart, install table, examples,
lineage commands, release integrity, and repository harness instructions.

## Next Check

```sh
make validate-docs
make verify
```
