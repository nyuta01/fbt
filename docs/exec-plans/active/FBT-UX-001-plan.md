# FBT-UX-001 Show actionable next steps for blocked and skipped work

## Observation

`fbt plan` explained dirty and blocked reasons, but users still had to infer the
next command. `fbt build` returned exit code `3` for blocked work but did not
show the blocked node details.

## Decision

Add planner-owned `next_steps` so both text and JSON output share the same
action guidance. Keep the guidance command-based and local-first: build missing
upstream artifacts, inspect blocked upstream artifacts, and inspect skipped
artifacts.

## Permanent Fix

Added `next_steps` to planner nodes, printed them as `next:` lines in
`fbt plan` and blocked/skipped `fbt build` output, covered blocked and skipped
cases in Go tests, and added a conformance assertion for blocked downstream
build guidance.

## Next Check

Run:

```sh
make verify
```
