# FBT-REL-005 Define Adapter Release Tags, Checksums, And Signature Workflow

## Observation

Official adapters install from source and have verification smoke, but they do
not yet have a concrete release operation comparable to the core CLI release
assets and checksums.

## Decision

Define the release workflow for nested adapter modules: module-scoped tags,
supported protocol metadata, install commands, checksums or signatures, and CI
validation before users are told a version is official.

## Permanent Fix

Document and, where practical, script the adapter release path without moving
provider dependencies into fbt core.

## Next Check

Run adapter install smoke, release docs scans, and `make verify`.
