# FBT-PILOT-001 Run Real-Workflow Pilots And Capture User Friction

## Observation

The current MVP has strong deterministic examples, smoke tests, docs, and
runner boundaries. The next meaningful UX signal should come from real
source directories rather than more speculative feature work.

The 2026-05-29 pilots ran daily QA operations and the external evidence-quality
report boundary through `doctor`, `plan`, `build`, and `artifact explain`.
Both workflows fit fbt's source-files-to-artifacts loop. The only concrete
first-user friction was copied examples: demo wrapper commands need
`FBT_SOURCE_ROOT` declared and set when the project is copied outside the
repository checkout.

## Decision

Run two or more realistic pilot workflows using real-ish operational source
sets. Capture where a first-time user hesitates: project setup, source
declarations, runner configuration, artifact inspection, daily operation,
quality checks, install, or retention.

Keep fbt core unchanged. Treat the copy-example issue as documentation/example
friction, not as a reason to add installer, scheduler, or runtime behavior to
core.

## Permanent Fix

Create pilot notes and turn observed friction into concrete backlog items or
small fixes. Do not expand fbt core merely because a pilot workflow needs a
scheduler, approval system, catalog, provider SDK, or storage backend.

Added `docs/pilots/2026-05-29-real-workflow-pilots.md` with command evidence,
observed output, and the friction backlog. Updated `examples/README.md` to
explain checked-out versus copied example execution and the required
`FBT_SOURCE_ROOT` runner environment.

## Next Check

`make verify` passed after the pilot notes and example guidance update.
