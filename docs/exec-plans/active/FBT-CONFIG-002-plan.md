# FBT-CONFIG-002 Reject Unknown YAML Fields With Actionable Diagnostics

Status: todo
Owner: agent
Updated: 2026-05-29

## Goal

Fail fast on misspelled, removed, or unsupported project YAML fields.

## Observation

The CLI now rejects unknown flags and extra arguments, but project/resource YAML
is still permissive in many places. A misspelled field can be silently ignored,
which is dangerous for a declarative build tool. The removed `review` field has
a targeted diagnostic, but that does not cover general schema drift.

## Decision

Add strict YAML field handling with diagnostics that include file, line,
resource name when available, and a concise hint. Use the current public config
contract as the source of truth, after `FBT-CONFIG-001` removes or reserves
inert fields.

## Permanent Fix

Pending. Expected permanent fix:

- Add unknown-field detection for `fs_project.yml` and resource files.
- Preserve compatibility only for explicitly documented aliases.
- Add conformance fixtures for misspelled top-level, transform, source, policy,
  eval, and runner fields.

## Next Check

Run:

```sh
make verify
```

Expected result: YAML typos fail with stable diagnostic codes instead of being
ignored.
