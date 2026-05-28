# dbt And DataChain Interoperability Example

Status: MVP-ready

Use this example when dbt or DataChain already produces the data artifacts, and
the missing step is a versioned human-facing file such as an operational brief,
release note, runbook update, or manual patch.

fbt does not run warehouse SQL or manage typed datasets here. It consumes the
files those tools wrote and records the receipt for the generated file artifact.

## Example Flow

```text
data/dbt/target/run_results.json
data/dbt/target/manifest.json
data/datachain/materialized_records.json
data/datachain/stats.json
  -> bin/build-data-tool-brief
  -> target/artifacts/data/data_tool_brief.md
  -> .fbt state and artifact lineage
```

Run:

```sh
fbt plan --project-dir examples/data_tool_interop --select data_tool_brief
fbt build --project-dir examples/data_tool_interop --select data_tool_brief
fbt artifact explain data_tool_brief --project-dir examples/data_tool_interop
```

The first plan should show one selected transform that will run. The build
creates:

```text
examples/data_tool_interop/target/artifacts/data/data_tool_brief.md
examples/data_tool_interop/.fbt/state/run_results.jsonl
examples/data_tool_interop/.fbt/state/artifact_versions.json
```

The generated brief names the dbt and DataChain files it used and includes
small evidence excerpts. `artifact explain` shows those same files as source
dependencies with their fingerprints.

## Production Shape

A production workflow usually runs these steps outside fbt first:

```sh
dbt build --target prod
python jobs/materialize_support_cases.py
```

Then fbt consumes their output directory:

```sh
fbt plan --select data_tool_brief
fbt build --select data_tool_brief
```

Keep ownership clear:

| Tool | Owns |
|---|---|
| dbt | Warehouse transformations, tests, and dbt artifacts. |
| DataChain | Dataset materialization and record selection. |
| fbt | Generated file artifact version, dependency fingerprints, checks, and lineage exports. |
