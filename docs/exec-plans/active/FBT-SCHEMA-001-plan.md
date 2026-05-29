# FBT-SCHEMA-001 Generate JSON Schema For Project Configuration

## Observation

The config parser now has strict field diagnostics and docs-backed semantics,
but users still lack editor completion and a standalone schema for external
validation.

The parser already contains the field allowlists, transform/eval type
registries, and artifact type registry needed to keep schema output aligned
with implementation.

## Decision

Generate or maintain a JSON Schema for `fs_project.yml` and resource files
from the implemented parser contract and documented project-config spec.

Generate two schemas: one for `fs_project.yml` and one for resource YAML files.
The generator reads Go source for artifact aliases, transform types, eval
types, and strict parser field sets, then checks the committed JSON artifacts
for drift.

## Permanent Fix

Add schema artifacts under `schemas/`, link them from docs, and add checks so
the schema cannot drift from parser-supported fields and reserved/removed
fields.

Added `schemas/project-config-v1.schema.json`,
`schemas/resource-file-v1.schema.json`, and
`scripts/generate-project-config-schema.py`. `make verify` now includes
`make project-config-schema-check`.

## Next Check

`make project-config-schema-check` and `make verify` passed.
