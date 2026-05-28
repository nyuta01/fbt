# FBT-UNIX-014 Remove Stale Non-Core Feature References From Specs

Status: todo  
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

Planned:

- Audit `docs/spec.md`, `docs/design-doc.md`,
  `docs/security-and-conformance-spec.md`, `docs/manifest-spec.md`, active
  plans, and research docs for stale non-core feature claims.
- Replace built-in docs/review/approval wording with artifact inspection and
  standard export wording.
- Keep `validate-docs` and `make verify` green.

## Next Check

Run:

```sh
rg -n "docs generation|docs generate|review gates|approval state|approval facets|fbt parse|fbt eval|fbt state|fbt runner" README.md docs apps/docs/src examples
make verify
```
