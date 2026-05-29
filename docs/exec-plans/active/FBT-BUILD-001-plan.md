# FBT-BUILD-001 Execute Selected Build DAGs In Dependency Order

Status: todo
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

Pending. Expected permanent fix:

- Add a topological build order over selected transforms.
- Re-plan or evaluate readiness after each upstream commit.
- Keep real blockers visible when an upstream is not selected, fails, or cannot
  satisfy confidence/eval requirements.
- Add CLI and conformance coverage for one command building a two-stage graph.

## Next Check

Run:

```sh
make verify
```

Expected result: one `fbt build --select tag:support` builds both the upstream
and downstream support artifacts when both are selected and no external blocker
remains.
