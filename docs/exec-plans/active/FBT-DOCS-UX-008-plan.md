# FBT-DOCS-UX-008 Explain README commands as checkpoints

## Observation

The README example had a clearer incident-runbook scenario, but the command
sequence still read like a list of CLI invocations. A first-time user could not
quickly tell what each command was for, what fbt did during that step, or what
new thing they received afterward.

## Decision

Rewrite the example command section as a sequence of user checkpoints:
preview, build, inspect, compare, and explain. For each command, state the
purpose, what fbt does, and what the user gets. Add short captured output
snippets where they clarify the lifecycle without making the README a full
terminal transcript.

## Permanent Fix

README now presents the incident runbook commands one step at a time. Each step
has a plain-language goal before the command and a concrete outcome after it.
The section includes actual `plan` output for the incident example and captured
offline quickstart output for build/history lifecycle signals. It closes
with the short mental model:
`plan` decides, `build` produces artifacts and receipts, and `artifact history`
explains later.

## Next Check

```sh
make verify
```
