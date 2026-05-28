# FBT-MVP-007 Implement runner discovery and diagnostics

Superseded note: the public `fbt runner` subcommands described here were
removed by `FBT-UNIX-013`. Runner diagnostics now surface through
`fbt doctor`, and runner authors use `make runner-conformance`.

## Observation

Transforms can name runner references, but core cannot yet resolve those
references from project config, plugin manifests, user-local plugins, or PATH.
The CLI also needs project-level runner diagnostics.

## Decision

Implement runner discovery baseline:

- resolve project `runners` entries with explicit commands first
- read project-local and user-local `fbt_plugin.yml` manifests
- fall back to the conventional PATH executable name
  `fbt-runner-<normalized-runner-name>`
- return deterministic diagnostics for missing commands and duplicate matches
- expose runner readiness through `fbt doctor` with human and JSON output

## Permanent Fix

Added plugin, runner discovery, doctor diagnostics, and CLI smoke tests for
project config, plugin manifest, PATH convention, and missing command behavior.

## Next Check

Run:

```sh
make verify
```

Result on 2026-05-28: pass.
