# FBT-BUILD-002 Persist Failed Build And Transform Receipts

Status: done
Owner: agent
Updated: 2026-05-29

## Goal

Make failed builds explainable through the same receipt model as successful
builds.

## Observation

fbt's value is the build receipt: what ran, what it read, what it wrote or tried
to write, which runner/eval/policy decisions were involved, and what changed.
The current build path appends successful transform run records after commit,
but early runner, policy, eval, cancellation, or protocol errors can return
before a failed transform record and failed invocation completion record are
persisted.

## Decision

Treat failure metadata as first-class build output. Persist safe failed-run
records without moving official artifact pointers. Do not store raw prompts,
raw model output, credentials, or unredacted tool payloads.

## Permanent Fix

- Append `transform_run` records for failed, cancelled, blocked, policy-denied,
  and eval-failed attempts when a run was started. Implemented for failed
  runner setup/capability checks, runner protocol errors, output-contract
  violations, policy denial, eval failure, and cancellation.
- Append `invocation_completed` with failed/cancelled/blocked status on every
  build exit path after `invocation_started`.
- Export failed-run spans/events through OTel where safe, including error type,
  error message, and an `exception` span event.
- Add conformance coverage proving failed runs explain what happened and do not
  update current artifact pointers.

## Next Check

Run:

```sh
make verify
```

Expected result: deterministic failure scenarios leave inspectable receipts and
do not corrupt official artifact state.

Latest result: passed.
