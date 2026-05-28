# FBT-H-003 Define runner discovery and plugin installation semantics

## Observation

Runner execution is central to `fbt`, but discovery was previously unresolved:
project config, `PATH`, and future plugin installation all appeared in docs
without a canonical precedence model.

## Decision

Pin MVP runner discovery as external process resolution:

- project `runners` entry with explicit `command`
- project-local plugin manifests under `plugins/*/fbt_plugin.yml`
- user-local plugin manifests under `${FBT_HOME:-~/.fbt}/plugins/*/fbt_plugin.yml`
- `PATH` convention using `fbt-runner-<normalized-runner-name>`

MVP does not download or install plugins. Future `fbt plugin install` is
reserved and must remain out-of-process.

## Permanent Fix

Added `docs/runner-discovery-spec.md`, linked it from project config, CLI, and
runner protocol docs, and removed runner discovery from the unresolved protocol
questions.

## Next Check

Run:

```sh
make verify
```

When runner code begins, add fake executable tests for precedence, ambiguity,
missing runner exit code `6`, and incompatible capability handling.
