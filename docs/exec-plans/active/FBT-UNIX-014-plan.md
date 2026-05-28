# FBT-UNIX-014 Remove Stale Non-Core Feature References From Specs

Status: done
Owner: agent
Updated: 2026-05-29

## Goal

Make the source-of-truth specs match the current Unix-style product boundary.

## Observation

After pruning `review`, `docs`, `parse`, `eval`, `state`, and `runner` from the
public command surface, several specs and historical docs still mention built-in
docs generation, approval state, review gates, or old diagnostic commands. That
creates drift between the actual product and the docs agents read before
changing code.

## Decision

Remove stale current-state claims from source-of-truth docs. Historical plans
may keep superseded context only when clearly labeled. The current product
boundary should be: build artifacts, record receipts, inspect artifacts, diff
versions, and export standard lineage/trace records.

## Permanent Fix

- Audited source-of-truth specs and active plans for stale docs/review/approval
  wording.
- Replaced current-state `docs generation`, approvals, review gates, and review
  command wording with docs-site build, confidence/upstream blocking, artifact
  inspection, and standard export language.
- Left explicit "outside core" and "superseded note" references in place where
  they document removed behavior rather than current behavior.
- Kept `validate-docs` and `make verify` green.

## Next Check

Run:

```sh
rg -n "docs generation|docs generate|review gates|approval state|approval facets|fbt parse|fbt eval|fbt state|fbt runner" README.md docs apps/docs/src examples
make verify
```

Expected result: any remaining hits are explicit outside-core or superseded
historical notes, not current product claims.
