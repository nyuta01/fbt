# FBT-STD-004 Evaluate OpenMetadata catalog export

## Observation

OpenMetadata integration was reserved in the standard export contract, but fbt
had not yet decided whether it should expose a direct `export openmetadata`
command or use the implemented OpenLineage export.

## Decision

Do not add a base `fbt export openmetadata` command. OpenMetadata is a catalog
and governance target reached through OpenLineage ingestion. Direct
OpenMetadata publishing belongs in an optional external integration only when a
team needs OpenMetadata-specific enrichment such as owners, domains, tags,
glossary terms, or custom properties.

## Permanent Fix

Added `docs/research/openmetadata-catalog-export-evaluation.md` and updated the
standard export spec, CLI reference, usage guide, visualization guide, README,
quality score, and task state. The docs now describe OpenMetadata's
OpenLineage ingestion path and explicitly keep fbt-native state as the source of
truth.

## Next Check

Run:

```sh
make verify
```
