# FBT-MVP-001 Implement project config and resource parser

## Observation

The repository has complete draft specs for `fs_project.yml`, resource YAML,
schema versioning, artifact aliases, path rules, and validation behavior, but
the Go implementation currently exposes only the CLI scaffold. Product parsing
must start with executable validation before graph or build behavior lands.

## Decision

Implement a small parser baseline:

- discover the project root from `fs_project.yml`
- require `config_version: 1`
- apply documented default resource paths
- load `.yml` and `.yaml` resource files from configured directories
- parse sources, artifacts, assets, transforms, policies, evals, and runners
- validate supported artifact aliases, names, references, asset paths, and
  output containment under `artifact_path`
- return structured diagnostics without adding runner or graph behavior yet

## Permanent Fix

Added executable parser tests covering required `config_version`, default
project config values, artifact type aliases, project discovery, valid resource
loading, output path containment, unresolved refs, and diagnostics errors.

## Next Check

Run:

```sh
make verify
```

Result on 2026-05-28: pass.
