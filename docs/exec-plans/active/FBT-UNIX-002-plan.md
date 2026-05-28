# FBT-UNIX-002 Make Artifact Explanation The Primary UX

Status: done
Owner: agent
Updated: 2026-05-29

## Goal

Make `fbt artifact explain` the focused command for understanding why an
artifact exists, will rebuild, will skip, or is blocked.

## Observation

`artifact explain` showed action, inputs, outputs, current version, previous
run, and next steps, but it did not summarize the decision or show enough
dependency detail for a user to see which source, asset, runner, policy, eval,
or upstream artifact affected the decision.

## Decision

Keep the primary loop small: `plan` previews all selected work, while
`artifact explain <artifact>` explains one artifact deeply. Explanations should
include a human decision sentence, dependency fingerprints, upstream artifact
requirements, current versions, dirty or blocked reasons, and the next command.

## Permanent Fix

- Added `decision` and `dependencies` to artifact explanation JSON.
- Printed source/asset/policy/eval/runner fingerprints and upstream artifact
  state in human `artifact explain` output.
- Added a run next-step so `plan` can point from dirty work to
  `fbt build --select <transform>`.
- Replaced the stale blocked-confidence `fbt eval` suggestion with
  `fbt artifact explain`.
- Updated CLI reference, docs site inspection guidance, and CLI tests.

## Next Check

Run:

```sh
make verify
```

Expected result: CLI tests and smoke coverage pass with richer explanation
output.
