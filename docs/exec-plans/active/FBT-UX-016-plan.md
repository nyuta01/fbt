# FBT-UX-016 Show Source-File Change Details In Dirty Explanations

## Observation

Daily source directories can accumulate many files. Current dirty explanations
tell users that a source changed, but not which concrete files were added,
changed, or removed.

## Decision

Expose source file-set deltas in `plan` and `artifact explain` without adding a
scheduler, watermark store, or partition engine. Keep the feature as
inspection metadata around existing source descriptors.

## Permanent Fix

Add machine-readable and human-readable source delta details so users can
understand why an artifact will rebuild from filesystem evidence alone.

## Next Check

Cover add/change/delete source-file cases in planner or CLI tests and run
`make verify`.
