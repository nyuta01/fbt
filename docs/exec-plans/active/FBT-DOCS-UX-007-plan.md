# FBT-DOCS-UX-007 Replace README example with incident runbook workflow

## Observation

The README example still did not make the practical job obvious. The support
manual scenario felt indirect, the transform details made the workflow look
more complex than necessary, and the command list did not clearly say what each
step gives the user.

## Decision

Replace the README example with the simpler incident-response workflow already
present in the repository: incident logs, response notes, and a postmortem
become an approved runbook. Remove YAML from the README example and explain the
recipe as a small table. For each command, state the result the user gets.

## Permanent Fix

README now explains the example as "incident notes to runbook": existing
incident evidence becomes a runbook plus a build receipt. The command flow is
shown one step at a time with the concrete outcome of `plan`, `build`,
`review`, and `artifact history`.

## Next Check

```sh
make verify
```
