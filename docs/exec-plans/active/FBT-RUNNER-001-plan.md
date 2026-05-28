# FBT-RUNNER-001 Pass complete execution context to protocol runners

## Observation

The runner protocol allowed inputs, assets, runner metadata, and state, but the
build lifecycle only sent transform identity, model, tools, policy, outputs,
and work directories. External provider or agent runners would have had to
reparse fbt project files or state to find input paths, current artifact
versions, prompt assets, and dirty/review context.

## Decision

Populate `fbt/runTransform` with execution context owned by core: resolved
source inputs, current artifact-version inputs, descriptors, semantic
descriptors, transform assets, runner config metadata, prior state, current
output pointers, plan dirty reasons, and review requirements.

## Permanent Fix

Added build helpers that construct complete protocol context from manifest and
state before runner invocation. Added `runner` to protocol run params and a
capturing fake-runner test that asserts source, ref, asset, runner config, and
state context are present.

## Next Check

Run:

```sh
make verify
```
