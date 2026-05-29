# FBT-REL-005 Define Adapter Release Tags, Checksums, And Signature Workflow

## Observation

Official adapters install from source and have verification smoke, but they do
not yet have a concrete release operation comparable to the core CLI release
assets and checksums.

Because official adapters are nested Go modules, their release identity must be
module-scoped rather than a root repository tag.

## Decision

Define the release workflow for nested adapter modules: module-scoped tags,
supported protocol metadata, install commands, checksums or signatures, and CI
validation before users are told a version is official.

Use signed annotated module-scoped tags plus Go module checksum verification as
the source-install baseline. Define `SHA256SUMS` and `cosign sign-blob` only
for future binary adapter archives.

## Permanent Fix

Document and, where practical, script the adapter release path without moving
provider dependencies into fbt core.

Updated `docs/release.md` and `docs/runner-adapters.md`, added
`.github/workflows/release-adapters.yml`, and added
`scripts/check-adapter-release-plan.py` behind `make adapter-release-plan-check`
and `make verify`.

## Next Check

`make adapter-release-plan-check` and `make verify` passed.
