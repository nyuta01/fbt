# fbt-runner-command

Official command adapter for fbt.

`fbt-runner-command` executes the `transform.command` argv from an fbt
`type: command` transform, then declares every file or directory written under
`work.outputs` as an output candidate.

The adapter is useful for Unix-style composition with existing scripts,
remark/Pandoc-style tools, dbt/DataChain output processors, and internal
converters while fbt keeps ownership of graph planning, state, artifact
commits, evals, and lineage.

## Development

From the repository root:

```sh
cd adapters/command
go test ./...
```

Protocol conformance from the repository root:

```sh
FBT_COMMAND_ADAPTER_DEFAULT_COMMAND="$PWD/adapters/command/testdata/write-output.sh" \
python3 tests/runner-conformance/run.py \
  --runner-command 'go run ./adapters/command/cmd/fbt-runner-command' \
  --transform-type command \
  --strict
```

`FBT_COMMAND_ADAPTER_DEFAULT_COMMAND` is only a conformance-test fallback for
requests that do not include `transform.command`; normal fbt projects should
declare the command in the transform.
