# FBT-UX-002 Add artifact explanation command for plan decisions

## Observation

Users could run `fbt plan`, but focusing on one artifact required scanning the
whole plan and correlating manifest/state records manually.

## Decision

Add `fbt artifact explain TARGET` as the focused explanation surface. It should
reuse planner decisions instead of inventing a separate explanation model, so
text and JSON output stay consistent with `fbt plan`.

## Permanent Fix

Implemented `fbt artifact explain TARGET` with producer transform, inputs,
outputs, current artifact version, previous run evidence, dirty reasons,
blocked reasons, and `next:` commands. Added CLI coverage and documented the
command in the CLI reference and usage guide.

## Next Check

Run:

```sh
make verify
```
