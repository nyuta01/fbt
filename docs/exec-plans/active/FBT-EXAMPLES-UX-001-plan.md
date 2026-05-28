# FBT-EXAMPLES-UX-001 Improve examples for first-time users

## Observation

The examples were technically valid, but they were hard to evaluate from a
user perspective. `knowledge_ops` was a runnable fixture but looked like a
business example, while the practical incident and support examples had useful
source data but terse READMEs that did not explain the job, inputs, output,
credential boundary, command outcomes, or generated receipts clearly. Project
doctor also printed an executable runner as `error RUNNER_COMMAND_OK` when a
separate runner environment variable was missing.

Reviewing the repeated-operation story exposed a more serious product gap:
adding a new file under a glob source did not mark the dependent transform
dirty, so daily source growth could be missed.

## Decision

Make `examples/` itself a routing surface. Clearly separate the offline control
plane fixture from practical external-runner workflows, explain each example's
user job, show concrete inputs and outputs, state which commands work without
credentials, and fix mixed-status doctor diagnostics so examples fail in an
understandable way. Treat local source file sets and file contents as part of
source fingerprinting so daily additions under globs make downstream transforms
dirty.

## Permanent Fix

Added `examples/README.md`, rewrote the three example READMEs around user
intent and command outcomes, and fixed doctor runner diagnostics so missing env
vars do not make unrelated successful checks look like failures. Source
fingerprints now include resolved file sets and file content fingerprints;
adding a file under a declared glob source causes `fbt plan` to rerun dependent
transforms with `source descriptor changed`.

## Next Check

```sh
make verify
```
