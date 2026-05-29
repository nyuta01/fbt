# FBT-CONFIG-003 Clarify local state and artifact directory controls

## Observation

The config surface exposed `state.backend`, `state.path`, `artifact_path`, and
the CLI exposed `--state-dir`. Without explicit wording and validation, users
could infer that fbt supports non-local state backends or that `--state-dir`
moves immutable artifact storage.

## Decision

Made MVP state backend semantics explicit: only `local` is supported,
`--state-dir` overrides the local receipt/state directory only, immutable
artifact snapshots stay under `.fbt/artifacts`, and `artifact_path` controls the
current logical outputs under `target/artifacts` by default.

## Permanent Fix

Rejected unsupported `state.backend` values and project-escaping `state.path`
values with actionable diagnostics. Added parser tests and updated the project
config, state, CLI, and docs-site references for the fixed local state model.

## Next Check

Done. Parser/config/CLI tests, docs/harness checks, and `make verify` pass.
