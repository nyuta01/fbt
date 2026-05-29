# FBT-CONTRACT-001 Clarify Artifact Contract And Eval Relationship

## Observation

Artifacts can declare `contract` metadata and transforms can declare evals, but
the boundary between artifact shape requirements, deterministic checks, and
runner-owned judge reports was still mostly implicit. Output contracts also
stopped at artifact resources and were not visible on manifest transform outputs
or runner protocol output declarations.

## Decision

Specify what fbt core understands about artifact contracts, which checks remain
deterministic evals, and which validation belongs to external runners. Avoid
turning core into a document validator or model judge. Treat contracts as
free-form metadata that fbt fingerprints, stores, and passes to runners.

## Permanent Fix

Align project-config docs, schemas, parser behavior, eval behavior, and examples
around a clear artifact contract model. Preserve output contracts in
`manifest.TransformOutput`, include them in `fbt/runTransform` output payloads,
and document that deterministic evals are the only core-owned validation path in
the MVP.

## Next Check

`make verify` must continue to pass. Future contract changes should stay
metadata-only unless a new deterministic validator is specified and tested.
