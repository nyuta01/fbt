# FBT-PILOT-001 Run Real-Workflow Pilots And Capture User Friction

## Observation

The current MVP has strong deterministic examples, smoke tests, docs, and
runner boundaries. The next meaningful UX signal should come from real
source directories rather than more speculative feature work.

## Decision

Run two or more realistic pilot workflows using real-ish operational source
sets. Capture where a first-time user hesitates: project setup, source
declarations, runner configuration, artifact inspection, daily operation,
quality checks, install, or retention.

## Permanent Fix

Create pilot notes and turn observed friction into concrete backlog items or
small fixes. Do not expand fbt core merely because a pilot workflow needs a
scheduler, approval system, catalog, provider SDK, or storage backend.

## Next Check

Run the pilot workflows, update docs/examples or backlog from the findings,
and run `make verify`.
