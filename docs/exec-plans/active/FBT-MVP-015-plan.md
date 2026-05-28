# FBT-MVP-015 Implement diff and lineage inspection

Superseded note: the public `fbt docs generate` command added by this task was
removed by `FBT-UNIX-013`. The supported public surfaces are now `fbt diff`,
`fbt artifact`, and standard exports.

## Observation

State now contains artifact versions, eval results, runner provenance, and
current pointers, but users cannot compare artifact versions from the CLI.

## Decision

Implement:

- raw text diff and Markdown heading-aware changed-section summaries
- `fbt diff TARGET [--against TARGET]`
- CLI and focused Go/smoke coverage

## Permanent Fix

Added focused diff package tests, CLI tests for `fbt diff`, and smoke coverage
for comparing generated artifacts after the build loop.

## Next Check

Run:

```sh
make verify
```

Result on 2026-05-28: pass.
