# FBT-MVP-007 Implement runner discovery and diagnostics

## Observation

Transforms can name runner references, but core cannot yet resolve those
references from project config, plugin manifests, user-local plugins, or PATH.
The CLI also lacks the specified `runner list`, `runner doctor`, and
`runner validate` diagnostics.

## Decision

Implement runner discovery baseline:

- resolve project `runners` entries with explicit commands first
- read project-local and user-local `fbt_plugin.yml` manifests
- fall back to the conventional PATH executable name
  `fbt-runner-<normalized-runner-name>`
- return deterministic diagnostics for missing commands and duplicate matches
- expose `fbt runner list|doctor|validate` with human and JSON output

## Permanent Fix

Added plugin, runner discovery, CLI runner diagnostics, and CLI smoke tests for
project config, plugin manifest, PATH convention, missing command, and runner
list behavior.

## Next Check

Run:

```sh
make verify
```

Result on 2026-05-28: pass.
