# FBT-UNIX-000 Register Unix-style product backlog

## Observation

Comparing fbt with dbt, remark, Pandoc, DVC, Snakemake, and DataChain showed
that fbt should not try to become a general transformation engine, scheduler,
dataset database, document converter, or workflow orchestrator.

The strongest product position is narrower: fbt should be the local-first
control plane that turns externally generated files into explainable,
versioned artifacts with lineage and build receipts.

## Decision

Register a Unix-style backlog that keeps fbt focused on one job:

```text
external tools generate files; fbt records, explains, versions, and exports the
artifact lifecycle
```

The backlog prioritizes better CLI explanation, stable batch-operation
patterns, and adapter examples for existing tools over adding provider SDKs,
document conversion, scheduling, or dataset storage to core.

## Permanent Fix

Added `FBT-UNIX-*` tasks to the structured feature list. The tasks make the
product boundary explicit, improve the core receipt/explain UX, define
daily source-window patterns, and add examples/adapters for existing tools such
as remark, Pandoc, dbt, and DataChain without expanding fbt core into those
tools' domains.

## Next Check

```sh
make verify
```
