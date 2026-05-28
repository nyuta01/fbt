# FBT-H-002 Define schema/versioning and artifact descriptor registries

## Observation

The repository had strong draft specs but left schema/versioning, artifact type
registry, descriptor canonicalization, and schema migration as open questions.
Those choices block parser, manifest, state, and artifact code because they
define persisted compatibility.

## Decision

Pin the first implementation baseline:

- `fs_project.yml` requires `config_version: 1`
- JSON outputs carry `metadata.fbt_schema_version`
- schema families use major-versioned URIs
- YAML artifact aliases map to `fbt.artifact.*.v1` descriptor identifiers
- file digests use SHA-256 over bytes
- directory digests use sorted canonical entries and reject symlink escapes
- semantic descriptors are optional and never replace raw descriptors
- artifact version IDs include the full SHA-256 digest token

## Permanent Fix

Added `docs/schema-and-versioning-spec.md`, linked it from the core specs, and
updated examples to include `config_version: 1`. The harness now requires the
schema/versioning spec.

## Next Check

Run:

```sh
make verify
```

When parser work begins, add tests for missing config version, unsupported
config version, descriptor canonicalization, and artifact type alias mapping.
