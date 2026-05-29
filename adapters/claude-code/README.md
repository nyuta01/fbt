# fbt-runner-claude-code

Official Claude Code adapter for fbt.

`fbt-runner-claude-code` wraps `claude -p` behind the fbt runner protocol. It
stages source files and assets under `work.root`, invokes Claude Code in
non-interactive bare mode, copies the final response into `work.outputs`, and
lets fbt commit the official artifact version.

The adapter defaults to the `claude` executable. Set `FBT_CLAUDE_CODE_COMMAND`
to override it for tests, custom installations, or hermetic conformance
fixtures.

## Development

```sh
cd adapters/claude-code
go test ./...
```

Agent-adapter conformance from the repository root:

```sh
FBT_CLAUDE_CODE_COMMAND="$PWD/adapters/claude-code/testdata/claude-code-fixture.sh" \
python3 tests/runner-conformance/run.py \
  --runner-command 'go run ./adapters/claude-code/cmd/fbt-runner-claude-code' \
  --transform-type agent \
  --strict \
  --agent-adapter
```

The script under `testdata/` is only a protocol fixture. It is not a user-facing
demo and is not used by normal projects.
