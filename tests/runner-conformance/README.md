# Runner Conformance

This directory contains a small black-box conformance harness for external fbt
runners. It starts a runner process over stdio JSON-RPC/JSONL, sends
`initialize`, sends a minimal `fbt/runTransform`, and verifies:

- protocol `0.1` is negotiated
- required transform and artifact capabilities are advertised
- the run returns `status: success`
- at least one output candidate is declared
- candidate paths exist and stay under `work.outputs`
- strict mode emits at least one `fbt/event` and one `fbt/outputCandidate`
  notification
- protocol responses and events do not leak the injected redaction marker
- source, logical artifact, and `.fbt/state` guard files are not modified by the
  runner

Run the default fixture against the source fake runner:

```sh
make runner-conformance
```

Run it against an external command:

```sh
FBT_RUNNER_CONFORMANCE_COMMAND='my-fbt-runner --flag value' make runner-conformance
```

For looser core-compatibility checks without strict notification requirements:

```sh
python3 tests/runner-conformance/run.py --runner-command 'my-fbt-runner'
```

Run the copyable adapter scaffold:

```sh
python3 tests/runner-conformance/run.py \
  --runner-command examples/runner_adapter_scaffold/bin/fbt-runner-example \
  --strict \
  --agent-adapter
```

Use `--agent-adapter` for adapters that wrap Codex CLI, Claude Code, Gemini CLI,
or similar external agents. It additionally requires an `fbt/event` attribute
named `fbt.adapter.staging_workspace` under `work.root` but outside
`work.outputs`, plus fail-closed policy mapping through
`fbt.adapter.policy_mode=fail_closed` or `fbt.adapter.policy_fail_closed=true`.

Add `--expect-policy-failure` when checking the negative path for a CLI-agent
adapter. The harness sends a policy the official adapters intentionally cannot
enforce, expects a structured JSON-RPC policy error, and verifies guarded source,
logical artifact, and `.fbt/state` files were not modified.

The JSON fixtures in `fixtures/` show the canonical minimal request shapes. The
harness generates temporary absolute work paths at runtime, so fixture paths are
illustrative rather than used as static golden input.
