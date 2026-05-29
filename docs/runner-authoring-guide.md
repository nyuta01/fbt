# fbt Runner Authoring Guide

Status: Draft  
Created: 2026-05-28  
Audience: authors of external fbt runners and CLI-agent adapters

## 1. What a Runner Is

An fbt runner is an external process that speaks stdio JSON-RPC/JSONL. It can be
written in Python, TypeScript, Go, Rust, shell, or any runtime that can read
JSON lines from stdin and write JSON lines to stdout.

For project users, a runner is just the command listed in `fs_project.yml`.
For runner authors, that command is responsible for the protocol and for any
provider SDK, agent CLI, converter, or internal service it wraps.

Core owns project parsing, planning, state, descriptors, policy/eval checks,
and official artifact commits. The runner owns transform execution and
candidate file generation.

## 2. Minimal Protocol Loop

A compatible runner must:

1. Read one JSON-RPC object per line from stdin.
2. Respond to `initialize` with protocol `0.1` and current capabilities.
3. Accept `initialized` as a notification.
4. Respond to `fbt/runTransform`.
5. Write generated files under `params.work.outputs`.
6. Return output candidates through `fbt/outputCandidate` notifications or
   `result.outputs`.
7. Emit JSON-RPC errors for fatal failures.
8. Stop when stdin closes or when cancellation is received.

Do not write human logs to stdout. Human debug logs belong on stderr.

## 3. Start From The Scaffold

The copyable starting point is:

```text
examples/runner_adapter_scaffold/
```

It contains a dependency-free Python runner that implements `initialize`,
`fbt/runTransform`, `fbt/event`, and `fbt/outputCandidate`. Replace the
`render_candidate()` function with the provider, CLI agent, converter, or
internal service call you need.

Check the scaffold with:

```sh
python3 tests/runner-conformance/run.py \
  --runner-command examples/runner_adapter_scaffold/bin/fbt-runner-example \
  --strict \
  --agent-adapter
```

Expected result:

```text
runner-conformance: ok
```

Keep stdout reserved for JSON-RPC messages. Put human debug logs on stderr, and
write final candidates only under `params.work.outputs`.

## 4. Initialize

The initialize response is authoritative for the current process. Static
manifests are useful for discovery, but build and doctor use runtime
capabilities to decide whether a selected transform is compatible.

Required capability fields:

```json
{
  "protocol": {
    "version": "0.1",
    "framing": "jsonl"
  },
  "capabilities": {
    "transform_types": ["llm", "agent", "command"],
    "artifact_types": ["markdown", "markdown_directory", "text", "directory"],
    "stream_events": true,
    "output_candidates": true,
    "supports_cancel": true
  }
}
```

Advertise only the transform and artifact types that the runner can actually
produce in this invocation. fbt rejects incompatible runners before execution.

## 5. Run Transform

`fbt/runTransform` provides:

- `transform`: name, type, fingerprint, and logical identity
- `runner`: configured runner metadata, environment names, and config
- `inputs`: resolved source paths and current upstream artifact versions
- `outputs`: declared output names, artifact types, and logical target paths
- `assets`: prompt/style/script asset paths and fingerprints
- `model`, `tools`, `policy`: execution contract metadata
- `state`: previous run, current outputs, and plan reasons
- `work`: scoped root, temp, and output directories

Runners should treat inputs, assets, state, and official target paths as
read-only. Write intermediate files under `work.temp` and final candidates under
`work.outputs`.

## 6. Output Candidates

Each candidate must include a declared output `name`, an `artifact_type`, and a
filesystem `path` under `work.outputs`.

```json
{
  "jsonrpc": "2.0",
  "method": "fbt/outputCandidate",
  "params": {
    "request_id": "run_001",
    "transform_run_id": "transform_run.runner_conformance",
    "outputs": [
      {
        "name": "output",
        "artifact_type": "markdown",
        "path": "/repo/.fbt/work/run_001/outputs/output",
        "declared_path": "target/artifacts/output.md"
      }
    ]
  }
}
```

Core computes descriptors and semantic descriptors. Runners may provide helpful
metadata, but runner-provided digests are advisory only.

## 7. Events and Redaction

Use `fbt/event` for progress, usage, safe tool-call summaries, warnings, and
debug metadata. Do not emit raw prompts, raw inputs, raw generated documents,
credentials, or unredacted tool arguments/results by default.

Useful event attributes follow OpenTelemetry GenAI names where they apply, for
example `gen_ai.provider.name`, `gen_ai.request.model`,
`gen_ai.usage.input_tokens`, and `gen_ai.usage.output_tokens`.

## 8. CLI-Agent Adapters

When wrapping Codex CLI, Claude Code, Gemini CLI, provider SDK agents, or an
internal agent launcher, the adapter process is the fbt runner.

The adapter should:

- run the agent in a staging workspace, scoped copy, or isolated work tree
- translate fbt policy into the agent's permission, sandbox, network, tool,
  timeout, and max-turn controls when available
- fail closed when policy cannot be represented safely
- copy final files into `work.outputs`
- emit only redacted events and declared output candidates

Recommended adapters should also emit a redacted `fbt/event` that proves the
boundary used for this run:

```json
{
  "method": "fbt/event",
  "params": {
    "event_type": "progress",
    "attributes": {
      "fbt.adapter.staging_workspace": "/.../.fbt/work/<run>/tmp/agent-staging",
      "fbt.adapter.policy_mode": "fail_closed"
    }
  }
}
```

The staging workspace must be under `params.work.root`, separate from
`params.work.outputs`, and safe to discard. If the adapter cannot map fbt's
policy into the external agent safely, it must return a JSON-RPC error instead
of running with broader permissions.

The external agent should not write directly to logical artifact paths,
immutable artifact storage, `.fbt/state`, or source paths during the normal fbt
build path.

## 9. Existing CLI Tool Adapters

For tools such as remark, Pandoc, converters, linters, and internal scripts,
prefer a thin command runner over implementing document logic in fbt core.

Project config points at the command runner:

```yaml
runners:
  - name: local.command
    type: command
    protocol: stdio_jsonrpc
    command: bin/fbt-command-runner
```

The transform declares the external argv:

```yaml
transforms:
  - name: pandoc_handbook
    type: command
    runner: local.command
    command: ["bin/run-pandoc-handbook"]
```

The command runner invokes that argv with `FBT_WORK_ROOT`, `FBT_WORK_TEMP`, and
`FBT_WORK_OUTPUTS` set. Project-local wrappers can set `FBT_COMMAND_WORKDIR`
when the runner process itself must start from another directory. The wrapper
script calls the existing tool and writes declared output candidates under
`FBT_WORK_OUTPUTS`; fbt records the artifact version, checks, and lineage after
the runner returns.

See `examples/markdown_toolchain` for remark-style Markdown normalization and a
Pandoc-style document conversion wrapper.

## 10. Conformance Check

Run the default harness against the source fake runner:

```sh
make runner-conformance
```

Run it against your runner:

```sh
FBT_RUNNER_CONFORMANCE_COMMAND='my-fbt-runner --flag value' make runner-conformance
```

For direct control:

```sh
python3 tests/runner-conformance/run.py \
  --runner-command 'my-fbt-runner' \
  --transform-type llm \
  --artifact-type markdown
```

The harness starts the runner, verifies `initialize`, sends a minimal
`fbt/runTransform`, and checks that declared candidate paths exist under
`work.outputs`. `--strict` also requires at least one progress event and one
`fbt/outputCandidate` notification.

For CLI-agent adapters, add `--agent-adapter`:

```sh
python3 tests/runner-conformance/run.py \
  --runner-command 'my-agent-adapter' \
  --strict \
  --agent-adapter
```

This mode injects a redaction marker into the temporary source and asset files,
creates guard files at the source path, logical artifact path, and `.fbt/state`,
then verifies that:

- protocol responses and events do not leak the marker
- the runner did not modify sources, official artifact paths, or `.fbt/state`
- a staging workspace was reported under `work.root` but outside `work.outputs`
- the adapter reported fail-closed policy mapping

Also check a negative policy path:

```sh
python3 tests/runner-conformance/run.py \
  --runner-command 'my-agent-adapter' \
  --strict \
  --agent-adapter \
  --expect-policy-failure
```

This mode sends a policy that the adapter is expected to reject before invoking
the external CLI. Passing the positive path alone only proves the adapter can
report a safe boundary; the negative path proves it fails closed.

For installed real adapters, use the opt-in matrix target:

```sh
FBT_RUNNER_ADAPTER_SMOKE_MATRIX='openai.responses|llm|markdown|fbt-runner-openai responses|OPENAI_API_KEY|false' \
make runner-adapter-smoke
```

The matrix runs conformance, a generated-project `doctor`, and a generated
project `plan` for each row. Add `FBT_RUNNER_ADAPTER_SMOKE_BUILD=1` only when
you intentionally want the smoke to call the real provider or agent and commit a
temporary artifact.

Expected successful conformance output is intentionally terse:

```text
runner-conformance: ok
```

## 11. Discovery Packaging

A runner can be referenced from project config, a plugin manifest, or a PATH
command convention. Use `command`, optional ordered `args`, optional `cwd`, and
declared `env` names. Do not rely on fbt passing the full ambient environment.

Provider SDKs, credentials, and heavyweight runtime dependencies belong in the
runner package, not in fbt core.

For recommended package names, plugin manifests, PATH conventions, and release
metadata, see [Runner Adapter Packaging](runner-adapters.md).
