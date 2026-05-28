# FBT-UNIX-008 Define Explicit Rebuild And Cache Controls

Status: done
Owner: agent
Updated: 2026-05-29

## Goal

Give users a small, explicit way to understand and override clean-skip behavior
without adding a general-purpose cache engine to fbt core.

## Observation

The planner already skipped clean transforms and rebuilt dirty ones, but users
had no first-class command flag for deliberate regeneration. A forced rebuild
also needed to preserve content-addressed immutable artifact versions when the
runner produced identical output.

## Decision

Add `--force` to `plan` and `build`. `plan --force` stays read-only and reports
`reason: forced rebuild`; `build --force` runs selected clean transforms while
still enforcing upstream, confidence, policy, and output-boundary checks.

## Permanent Fix

- Added `Force` to planner and build inputs.
- Added `--force` parsing for `plan` and `build`.
- Preserved immutable artifact-version semantics by reusing an existing
  content-addressed version when a forced rebuild produces identical output.
- Added planner, build, CLI, and knowledge-loop smoke coverage.
- Documented force behavior in CLI, usage, spec, and docs-site pages.

## Next Check

Run:

```sh
make verify
```

Expected result: clean selected transforms skip by default, run with
`--force`, and still fail normally for blocking or policy conditions.
