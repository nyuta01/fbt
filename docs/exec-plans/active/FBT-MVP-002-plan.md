# FBT-MVP-002 Implement manifest graph and selectors

## Observation

The parser now returns validated project resources, but downstream planning,
state comparison, docs, and CLI work need canonical resource IDs, dependency
maps, deterministic JSON, and selector evaluation.

## Decision

Implement a manifest baseline on top of the parser:

- convert parser resources into canonical manifest resource maps
- generate `source`, `artifact`, `transform`, `transform_asset`, `policy`,
  `eval`, and `runner` unique IDs
- build parent and child maps for transform inputs, outputs, assets, policies,
  evals, and runners
- provide deterministic JSON serialization
- add selector support for name, tag, path, resource_type, parent, and child

## Permanent Fix

Added manifest and selector unit tests covering canonical resource IDs,
parent/child maps, deterministic JSON serialization, selector methods, and
selector union evaluation.

## Next Check

Run:

```sh
make verify
```

Result on 2026-05-28: pass.
