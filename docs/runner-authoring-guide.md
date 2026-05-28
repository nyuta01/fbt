# fbt Runner Authoring Guide

Status: Draft  
Created: 2026-05-28  
Audience: authors of external fbt runners and CLI-agent adapters

## 1. What a Runner Is

An fbt runner is an external process that speaks stdio JSON-RPC/JSONL. It can be
written in Python, TypeScript, Go, Rust, shell, or any runtime that can read
JSON lines from stdin and write JSON lines to stdout.

Core owns project parsing, planning, state, descriptors, policy/eval/review
checks, and official artifact commits. The runner owns transform execution and
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

## 3. Initialize

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

## 4. Run Transform

`fbt/runTransform` provides:

- `transform`: name, type, fingerprint, and logical identity
- `runner`: configured runner metadata, environment names, and config
- `inputs`: resolved source paths and current upstream artifact versions
- `outputs`: declared output names, artifact types, and logical target paths
- `assets`: prompt/style/script asset paths and fingerprints
- `model`, `tools`, `policy`: execution contract metadata
- `state`: previous run, current outputs, plan reasons, and review context
- `work`: scoped root, temp, and output directories

Runners should treat inputs, assets, state, and official target paths as
read-only. Write intermediate files under `work.temp` and final candidates under
`work.outputs`.

## 5. Output Candidates

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

## 6. Events and Redaction

Use `fbt/event` for progress, usage, safe tool-call summaries, warnings, and
debug metadata. Do not emit raw prompts, raw inputs, raw generated documents,
credentials, or unredacted tool arguments/results by default.

Useful event attributes follow OpenTelemetry GenAI names where they apply, for
example `gen_ai.provider.name`, `gen_ai.request.model`,
`gen_ai.usage.input_tokens`, and `gen_ai.usage.output_tokens`.

## 7. CLI-Agent Adapters

When wrapping Codex CLI, Claude Code, Gemini CLI, provider SDK agents, or an
internal agent launcher, the adapter process is the fbt runner.

The adapter should:

- run the agent in a staging workspace, scoped copy, or isolated work tree
- translate fbt policy into the agent's permission, sandbox, network, tool,
  timeout, and max-turn controls when available
- fail closed when policy cannot be represented safely
- copy final files into `work.outputs`
- emit only redacted events and declared output candidates

The external agent should not write directly to logical artifact paths,
immutable artifact storage, `.fbt/state`, or source paths during the normal fbt
build path.

## 8. Conformance Check

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

## 9. Discovery Packaging

A runner can be referenced from project config, a plugin manifest, or a PATH
command convention. Use `command`, optional ordered `args`, optional `cwd`, and
declared `env` names. Do not rely on fbt passing the full ambient environment.

Provider SDKs, credentials, and heavyweight runtime dependencies belong in the
runner package, not in fbt core.

For recommended package names, plugin manifests, PATH conventions, and release
metadata, see [Runner Adapter Packaging](runner-adapters.md).
