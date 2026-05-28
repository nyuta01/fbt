# FBT-UNIX-001 Clarify The Single-Purpose Product Boundary

Status: done
Owner: agent
Updated: 2026-05-29

## Goal

Make fbt's Unix-style product boundary explicit in the first documents users
read.

## Observation

The docs already described fbt as a local-first file build tool, but adjacent
tool boundaries were scattered. A new user could still infer that fbt might
grow into a dbt, DataChain, DVC, Snakemake, remark, Pandoc, scheduler, provider
SDK, artifact store, or catalog replacement.

## Decision

Keep fbt's one job narrow: generated file artifact receipts. Adjacent tools
remain external systems that fbt composes with through files, runners, or
standard exports.

## Permanent Fix

- Updated README, design doc, core spec, usage guide, and docs-site
  introduction to name the non-replacement boundary.
- Removed a duplicated docs-site ownership bullet while clarifying the
  lifecycle fbt does own.
- Registered this plan as the task's durable handoff record.

## Next Check

Run:

```sh
make verify
```

Expected result: docs validation and docs-site build pass with the narrower
boundary language.
