# FBT-UNIX-011 Remove Built-In Review From Core

Status: done  
Owner: agent  
Updated: 2026-05-29

## Goal

Remove fbt-native review and approval behavior so the tool stays aligned with
the Unix-style "do one thing well" boundary: build versioned filesystem
artifacts and record receipts.

## Observation

Review had become a first-class fbt feature through CLI commands, local approval
state, review gates, `human_review` evals, approval-related lineage facets, and
example workflows. That made fbt look like a review/approval workflow product,
which duplicates responsibilities that already fit better in Git, PRs, CI,
release tooling, ticket systems, or knowledge-base publishing flows.

## Decision

Remove review from core instead of making it optional. Keep confidence and eval
requirements because they are build-quality checks; remove human approval state
and review commands because they are workflow decisions outside fbt.

## Permanent Fix

- Deleted `internal/approval`.
- Removed `fbt review` from CLI routing, docs, examples, and smoke flows.
- Removed approvals from state, planner, build, docs generation, eval, lineage,
  OpenLineage facets, runner protocol fixtures, templates, and conformance.
- Removed review fields from practical example policies/transforms.
- Updated source-of-truth docs, docs site pages, README, examples, quality
  score, failure log, and progress handoff.
- Added smoke/test coverage that asserts `fbt review` is an unknown command.

## Next Check

Run:

```sh
make verify
```

Expected result: all checks pass with no built-in review or approval workflow in
the command surface, state model, examples, or docs.
