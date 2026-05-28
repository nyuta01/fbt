# dbt And DataChain Interop Example

This example shows fbt consuming outputs from data tools without becoming one.

```text
dbt target/run_results.json + dbt target/manifest.json
DataChain materialized records + DataChain stats
  -> target/artifacts/data/data_tool_brief.md
  -> fbt receipt with source fingerprints and lineage
```

Use this pattern when dbt or DataChain already owns data transformation, and
the missing artifact is an explainable operational brief, release note, or
manual update for humans.

## Run It

```sh
fbt plan --project-dir examples/data_tool_interop --select data_tool_brief
fbt build --project-dir examples/data_tool_interop --select data_tool_brief
fbt artifact explain data_tool_brief --project-dir examples/data_tool_interop
```

The checked-in wrapper is deterministic. In a real project, replace
`bin/build-data-tool-brief` with a script or runner that reads actual dbt and
DataChain outputs and writes the declared Markdown candidate under
`FBT_WORK_OUTPUTS`.

fbt does not transform warehouse tables or manage typed datasets here. It
records which dbt/DataChain output files produced the generated brief.
