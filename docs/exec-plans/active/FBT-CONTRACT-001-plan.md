# FBT-CONTRACT-001 Clarify Artifact Contract And Eval Relationship

## Observation

Artifacts can declare `contract` metadata and transforms can declare evals, but
the boundary between artifact shape requirements, deterministic checks, and
runner-owned judge reports is still mostly implicit.

## Decision

Specify what fbt core understands about artifact contracts, which checks remain
deterministic evals, and which validation belongs to external runners. Avoid
turning core into a document validator or model judge.

## Permanent Fix

Align project-config docs, schemas, parser behavior, eval behavior, and examples
around a clear artifact contract model.

## Next Check

Add focused examples and schema/parser checks that show the contract/eval
boundary, then run `make verify`.
