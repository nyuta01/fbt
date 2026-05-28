# FBT-UNIX-007 Complete Composable Graph Selection UX

Status: done
Owner: agent
Updated: 2026-05-29

## Goal

Make transform selection feel familiar to users of DAG-oriented CLIs by
supporting upstream and downstream graph expansion from the normal `--select`
surface.

## Observation

The CLI accepted names, tags, paths, resource types, and named selectors, but
leading and trailing `+` were only stripped from names. Users could not ask for
"this transform and its upstream/downstream transforms" in one expression.

## Decision

Implement `+target`, `target+`, and `+target+` in the graph package and use the
same transform-selection function from both `plan` and `build`. Expand through
the full resource graph, but return only transform IDs to execution.

## Permanent Fix

- Added graph expansion for upstream and downstream transform selection.
- Added ambiguity and invalid graph-selector errors.
- Reused the shared graph selection function from CLI and build paths.
- Added graph tests and knowledge-loop smoke assertions for upstream and
  downstream selection.
- Documented graph operators in the CLI reference.

## Next Check

Run:

```sh
make verify
```

Expected result: `+weekly_support_insights` plans the upstream case-summary
transform, and `case_summaries+` plans the downstream weekly-insights
transform after the upstream artifact exists.
