# FBT-MVP-015 Implement diff and docs generation

## Observation

State now contains artifact versions, eval results, approvals, runner
provenance, and current pointers, but users cannot compare artifact versions or
generate a static project report from that state.

## Decision

Implement:

- raw text diff and Markdown heading-aware changed-section summaries
- `fbt diff TARGET [--against TARGET|last-approved]`
- static Markdown docs generation with graph resources, artifact versions,
  eval results, approvals, and review/confidence state
- CLI and focused Go/smoke coverage

## Permanent Fix

Added focused diff/docs package tests, CLI tests for `fbt diff` and
`fbt docs generate`, and extended the knowledge-loop smoke to generate static
docs after the review/build loop.

## Next Check

Run:

```sh
make verify
```

Result on 2026-05-28: pass.
