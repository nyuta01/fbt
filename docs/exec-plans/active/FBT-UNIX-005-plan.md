# FBT-UNIX-005 Add dbt And DataChain Interoperability Examples

Status: done
Owner: agent
Updated: 2026-05-29

## Goal

Show that fbt composes with dbt and DataChain without replacing warehouse
transformation or dataset materialization tools.

## Observation

The product boundary said dbt and DataChain should remain adjacent tools, but
users had no runnable example showing what that composition looks like in an
fbt project.

## Decision

Model dbt and DataChain outputs as ordinary fbt sources. Use a deterministic
command transform to produce a Markdown brief from those files, so the smoke
test proves fbt records the artifact receipt and lineage without adding dbt or
DataChain dependencies to core.

## Permanent Fix

- Added `examples/data_tool_interop` with dbt run artifacts, DataChain job
  outputs, policy, transform, and a command runner wrapper.
- Added practical smoke coverage that plans, builds, and explains the
  generated brief.
- Added a dedicated interoperability example doc and linked the example from
  usage, docs-site, and examples index guidance.

## Next Check

Run:

```sh
make verify
```

Expected result: practical examples smoke builds `data_tool_brief` and
`artifact explain` lists both dbt and DataChain source dependencies.
