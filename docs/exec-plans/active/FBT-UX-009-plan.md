# FBT-UX-009 Tighten CLI help and command descriptions

## Observation

The command surface was intentionally small, but several help entries still
used generic descriptions. Artifact subcommands all said "Inspect artifacts",
standard export subcommands said only "Export openlineage" or "Export otel",
and global flag descriptions did not explain where `--select` and `--state-dir`
matter.

## Decision

Improved Cobra help text, CLI reference, usage guide wording, and the CLI smoke
expectation without adding new command surfaces. The Unix-style command set
stays intact and each user-facing command now describes what it returns.

## Permanent Fix

Added tests that assert command help exposes specific artifact, export, and
flag descriptions so future generic text regressions fail locally. The CLI
smoke now checks the product description from the current root help.

## Next Check

Done. `go test ./internal/cli`, `make validate-docs`, and `make verify` pass.
