# FBT-UX-015 Make The First Own-Files Success Path Self-Service

## Observation

fbt's examples and docs now explain the source/instruction/runner/artifact
model, but a first-time user may still need to infer how to replace the sample
project with their own directory of files. The current successful path is clear
for repository examples, less clear for "I have this folder; make my first
artifact."

## Decision

Improve the first-user path without expanding core scope. The goal is a
self-service loop that starts from a user's own files, declares minimal
sources, adds one instruction/format asset, configures a runner, builds one
artifact, and inspects the receipt.

## Permanent Fix

Add or revise README/docs/examples so the first custom project path is
copy-pasteable and verifiable. Prefer a small template or example over a new
CLI wizard unless existing init behavior already supports it cleanly.

## Next Check

Run docs scans for own-files guidance, any new example smoke, docs-site build,
and `make verify`.
