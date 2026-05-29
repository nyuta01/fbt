# FBT-DOCS-DRIFT-002 Remove stale review-gates language from docs site assets

## Observation

The source docs now correctly say review and approval are outside fbt core, but
the public docs OG image still contains `review gates`. This is visible
stale-current-state language in a public asset, not only a historical note.

## Decision

Treat generated/static docs assets as part of the docs source of truth. Remove
current-state review/approval claims from public docs assets and add a drift
guard that scans those assets, while preserving historical references in
exec-plan and failure-log files.

## Permanent Fix

Update the OG image wording to match the current product boundary. Extend the
drift check with a public-docs asset scan for stale `review gates` or equivalent
current-state approval phrasing.

## Next Check

Run the stale-language scan, docs-site build, and `make verify`.
