# FBT-STATE-002 Define Local State And Artifact Retention Hygiene

Status: done
Owner: agent
Updated: 2026-05-29

## Goal

Define the smallest safe answer for long-running local projects whose artifact
history grows every day.

## Observation

Immutable artifact versions are central to fbt's value. Daily projects that
process many source files will accumulate `.fbt/artifacts`, run results,
evaluation results, and policy decisions. The current design explains
immutability but does not define retention, archive, or pruning behavior.

## Decision

Design retention as local state hygiene, not as a metadata database, artifact
store, scheduler, or hosted service. The default should remain safe and
inspectable. Any destructive cleanup must be explicit and receipt-aware.

## Permanent Fix

MVP retention is now explicit and non-destructive. The policy is `keep_all`:
fbt does not automatically delete artifact versions, run results, eval results,
or policy decisions.

Added read-only inspection through `fbt artifact retention`. It reports state
bytes, immutable artifact bytes, run-record count, artifact-version count,
current-version count, historical-version count, and missing immutable storage
references. It removes no files.

Added `state.BuildRetentionReport` with unit coverage and product conformance
coverage for the CLI output. Docs now tell high-volume users to archive
`.fbt/state/` and `.fbt/artifacts/` together with external tools. No destructive
prune command is exposed in MVP; any future prune must default to dry-run,
preserve current pointers, record cleanup receipts, and have conformance before
it can remove files.

## Next Check

Run:

```sh
make verify
```

Latest targeted results: `go test ./internal/state ./internal/cli` and
`make conformance` passed. Final gate: `make verify` passed with the read-only
retention command.
