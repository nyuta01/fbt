# FBT-BUILD-001 Execute Selected Build DAGs In Dependency Order

Status: done
Owner: agent
Updated: 2026-05-29

## Goal

Make `fbt build` complete all selected runnable transforms in dependency order
within one invocation.

## Observation

`fbt build --select tag:support` can select both an upstream and downstream
transform, but the downstream transform is planned as blocked when the upstream
artifact does not already exist in the state snapshot. The current build loop
then executes the upstream transform and returns with the downstream still
blocked, forcing the user to run another command. That is surprising for a build
tool with a declared artifact graph.

## Decision

Teach the planner/build lifecycle to distinguish external blockers from
in-invocation dependencies. A selected downstream transform should wait for the
selected upstream transform, then run after the upstream artifact is committed
and confidence requirements are satisfied. Preserve fail-fast behavior until a
separate task deliberately implements broader execution policy.

## Permanent Fix

- Added dependency-ordered planning for selected transforms based on artifact
  producer/consumer edges.
- Missing upstream artifacts no longer block a transform when the selected graph
  also includes the upstream producer.
- Planned upstream runs propagate a dirty reason to selected downstream
  transforms so the graph can complete in one invocation.
- `build` rechecks runtime blockers before each transform using the latest
  in-memory state, so confidence or missing-input blockers still stop the
  downstream run when they remain after upstream work.
- Added planner/build tests and smoke/conformance coverage for a two-stage graph
  built by one command.

## Next Check

Run:

```sh
make verify
```

Expected result: one `fbt build --select tag:support` builds both the upstream
and downstream support artifacts when both are selected and no external blocker
remains.
