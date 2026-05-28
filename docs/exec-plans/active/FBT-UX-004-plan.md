# FBT-UX-004 Superseded Review Inspection With Artifact Inspection

Superseded note: the `fbt review` command originally implemented by this task
was removed by `FBT-UNIX-011`.

## Observation

Users needed a single inspection surface showing the selected artifact version,
output paths, digest, runner/model, generating run, and inspection commands.

## Decision

Keep inspection in the artifact and diff command surfaces.

## Permanent Fix

Current supported behavior is `fbt artifact show`, `fbt artifact history`,
`fbt artifact explain`, and `fbt diff`. Review/approval guidance was removed
from core by `FBT-UNIX-011`.

## Next Check

Run:

```sh
make verify
```
