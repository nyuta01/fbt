# FBT-CONFIG-001 Remove Or Implement Inert Config Fields

Status: done
Owner: agent
Updated: 2026-05-29

## Goal

Ensure every accepted project config field either has real behavior or is
explicitly reserved with actionable diagnostics.

## Observation

`execution.max_workers`, `execution.fail_fast`, `defaults.cache`,
`defaults.confidence`, and transform-level `cache` appear in the project
contract. Some are decoded and fingerprinted but do not currently change
execution behavior. In a declarative CLI, placebo config is worse than missing
config because users will trust settings that do not act.

## Decision

Audit the config surface through the Unix lens: keep only the smallest set of
controls needed for the build-receipt model. Implement fields that are essential
now. Mark future fields as reserved or remove them from the public contract.

## Permanent Fix

- Reserved `execution.max_workers`, `execution.fail_fast`, `defaults.cache`,
  `defaults.confidence`, and transform-level `cache` with
  `CONFIG_FIELD_RESERVED` diagnostics that include file, line, resource, and
  a hint.
- Removed the no-op fields from the public config structs and transform
  manifest resource.
- Aligned examples and the project-config spec so public YAML no longer
  presents hidden cache/default/parallel controls.
- Kept the explicit rebuild model: `plan --force` previews and `build --force`
  rebuilds selected transforms without bypassing dependency, confidence,
  policy, or output-boundary checks.

## Next Check

Run:

```sh
make verify
```

Expected result: project YAML no longer accepts no-op controls as if they were
active behavior.

Latest result: passed.
