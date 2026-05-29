# fbt-runner-codex-cli

Official Codex CLI adapter for fbt.

`fbt-runner-codex-cli` wraps `codex exec` behind the fbt runner protocol. It
stages source files and assets under `work.root`, invokes Codex in
non-interactive mode, copies the final response into `work.outputs`, and lets
fbt commit the official artifact version.

The adapter defaults to the `codex` executable. Set `FBT_CODEX_CLI_COMMAND` to
override it for tests, custom installations, or hermetic conformance fixtures.

## Development

```sh
cd adapters/codex-cli
go test ./...
```

Agent-adapter conformance from the repository root:

```sh
FBT_CODEX_CLI_COMMAND="$PWD/adapters/codex-cli/testdata/codex-cli-fixture.sh" \
python3 tests/runner-conformance/run.py \
  --runner-command 'go run ./adapters/codex-cli/cmd/fbt-runner-codex-cli' \
  --transform-type agent \
  --strict \
  --agent-adapter
```

The script under `testdata/` is only a protocol fixture. It is not a user-facing
demo and is not used by normal projects.
