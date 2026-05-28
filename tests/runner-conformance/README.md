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

The JSON fixtures in `fixtures/` show the canonical minimal request shapes. The
harness generates temporary absolute work paths at runtime, so fixture paths are
illustrative rather than used as static golden input.
