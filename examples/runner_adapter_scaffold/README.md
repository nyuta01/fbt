# Runner Adapter Scaffold

This is the smallest useful starting point for an external fbt runner adapter.
It has no provider SDK dependency and uses only the Python standard library.

Copy this directory when you want to wrap one executable, model provider, CLI
agent, converter, or internal service behind the fbt runner protocol.

## What To Replace

`bin/fbt-runner-example` already implements the protocol loop:

```text
stdin JSON-RPC initialize
stdin JSON-RPC fbt/runTransform
  -> write files under params.work.outputs
  -> emit fbt/event and fbt/outputCandidate
stdout JSON-RPC result
```

Replace `render_candidate()` with the provider, agent, or tool call you need.
Keep the rest of the boundary:

- read inputs and assets from the paths fbt sends
- run external CLI agents from a staging workspace under `work.root`
- write only under `work.outputs`
- emit redacted events only
- fail closed when fbt policy cannot be mapped safely
- return declared output candidates
- keep credentials in the runner environment, not fbt state

## Check It

From the repository root:

```sh
python3 tests/runner-conformance/run.py \
  --runner-command examples/runner_adapter_scaffold/bin/fbt-runner-example \
  --strict \
  --agent-adapter
```

Expected output:

```text
runner-conformance: ok
```

To exercise the same adapter through the installed-adapter smoke matrix and a
temporary fbt project:

```sh
FBT_RUNNER_ADAPTER_SMOKE_MATRIX='scaffold.agent|agent|markdown|examples/runner_adapter_scaffold/bin/fbt-runner-example||true' \
FBT_RUNNER_ADAPTER_SMOKE_BUILD=1 \
make runner-adapter-smoke
```

## Package Shape

An adapter package can ship this shape:

```text
bin/fbt-runner-example
fbt_plugin.yml
README.md
```

The plugin manifest is optional metadata. fbt still validates the running
process through `initialize`, so runtime capabilities are authoritative.
