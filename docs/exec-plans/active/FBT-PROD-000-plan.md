# FBT-PROD-000 Plan

## Task

Turn the remaining production-operation concerns into explicit backlog tasks
without expanding fbt core beyond its file build tool boundary.

## Observation

The daily support knowledge reference loop now shows the intended production
composition, but several concerns still require concrete follow-up work before
a team can claim production readiness: live runners, source-window semantics,
quality gates, retention/archive handoff, approval and publishing integration,
security execution profiles, and CI authority.

## Decision

Add P0/P1 tasks that resolve each concern through examples, docs, smoke tests,
adapter hardening, and CI workflows. Keep scheduling, review, publishing,
provider retries, and storage lifecycle outside fbt core unless a future spec
proves core behavior is necessary.

## Permanent Fix

The structured backlog now names every production-readiness concern with an
owner, priority, dependency chain, expected paths, and verification intent.
Quality-score rows point at open production-readiness tasks so the risks stay
visible until implemented.

## Next Check

- `make verify`
