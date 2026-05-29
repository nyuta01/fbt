# FBT-SCHEMA-001 Generate JSON Schema For Project Configuration

## Observation

The config parser now has strict field diagnostics and docs-backed semantics,
but users still lack editor completion and a standalone schema for external
validation.

## Decision

Generate or maintain a JSON Schema for `fs_project.yml` and resource files
from the implemented parser contract and documented project-config spec.

## Permanent Fix

Add schema artifacts under `schemas/`, link them from docs, and add checks so
the schema cannot drift from parser-supported fields and reserved/removed
fields.

## Next Check

Validate sample projects against the schema, run config parser tests, and run
`make verify`.
