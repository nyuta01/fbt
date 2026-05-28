# FBT-UNIX-010 Use Standard Visualization Backends In Examples

Status: done
Owner: agent
Updated: 2026-05-29

## Goal

Keep visualization out of fbt core and make the example path concrete enough
that users can send fbt exports to real standard backends.

## Observation

The docs already said to use OpenLineage and OTel exports, but the docs-site
visualization page still centered an abstract product diagram. Users need
copy-paste export and ingestion recipes, not invented fbt graph imagery.

## Decision

Replace diagram-led explanation with reproducible commands. Add an example that
creates exports from the offline support template and shows how to post them to
Marquez or an OTLP HTTP endpoint. Make screenshots a backend-capture rule.

## Permanent Fix

- Added `examples/standard_visualization` with standard backend recipes.
- Updated the standard visualization guide with a reproducible local export
  flow and screenshot rule.
- Rewrote the docs-site visualization page around commands and standard
  backends instead of a custom fbt graph image.
- Linked standard visualization from README and examples index.

## Next Check

Run:

```sh
make verify
```

Expected result: docs validation and docs-site build pass without requiring
Marquez, Jaeger, Tempo, Grafana, OpenMetadata, or a custom fbt visualization
service in the default gate.
