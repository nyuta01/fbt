# fbt Runner Adapter Packaging

Status: Draft  
Created: 2026-05-28  
Audience: maintainers and authors of optional provider or agent integrations

## 1. Scope

`fbt` core stays provider-free and runtime-free. OpenAI, Anthropic, Gemini,
Codex CLI, Claude Code, local model servers, document converters, SaaS
connectors, and internal agents belong in optional out-of-process runner
packages.

For users, the mental model is simple: a runner is an external command that
speaks the fbt runner protocol. The command may call a model provider, start an
agent CLI, run a converter, or invoke an internal service. fbt only cares that
the command advertises capabilities, reads the fbt request, writes output
candidates under `work.outputs`, and returns protocol messages.

An adapter package is responsible for:

- shipping one or more executable runner commands
- owning provider SDK, agent runtime, and converter dependencies
- reading credentials from its own environment
- speaking the fbt stdio JSON-RPC runner protocol
- advertising runtime capabilities through `initialize`
- writing only output candidates under `work.outputs`
- documenting its own provider-specific setup and limits

fbt core is responsible only for discovery, invocation, capability validation,
state, policy/eval checks, descriptors, and official commits.

For the research-backed official adapter package design and maintenance model,
see [Official Runner Adapter Design Report](research/official-runner-adapter-design-report.md).

The agreed repository strategy is monorepo with nested modules:

```text
sdk/go/                provider-free runner SDK
adapters/command/      official command adapter
adapters/openai/       official OpenAI adapter
adapters/codex-cli/    official Codex CLI adapter
adapters/claude-code/  official Claude Code adapter
```

The root `go.mod` remains fbt core only. Adapter modules own their provider or
CLI dependencies. Future SDKs such as `sdk/python` or `sdk/typescript` are
possible because the protocol is language-neutral JSON-RPC over stdio, but the
protocol spec and conformance suite remain the source of truth.

`adapters/command`, `adapters/openai`, `adapters/codex-cli`, and
`adapters/claude-code` are the first official nested adapter modules. The
command adapter executes a declared command transform argv and reports files
written under `work.outputs` as output candidates. The OpenAI adapter calls the
Responses API for `type: llm` transforms and keeps `OPENAI_API_KEY` outside fbt
core. The Codex CLI and Claude Code adapters wrap existing CLI-agent
executables through staging workspaces and fail-closed policy markers.

## 2. Minimal Scaffold

Adapter is the authoring/package word for a runner command plus its dependency
and setup wrapper. Start from `examples/runner_adapter_scaffold` when building
a new adapter. It contains:

```text
bin/fbt-runner-example
fbt_plugin.yml
README.md
```

The runner is a dependency-free Python stdio JSON-RPC process. It passes strict
conformance and marks the one function an adapter author should replace with a
real provider, agent, converter, or service call.

```sh
python3 tests/runner-conformance/run.py \
  --runner-command examples/runner_adapter_scaffold/bin/fbt-runner-example \
  --strict \
  --agent-adapter
```

The repository also keeps source-checkout adapter examples under
`examples/runner_adapters/` and test-only protocol fixtures under
`tests/runner_fixtures/`. There is intentionally no top-level `runners/`
directory, because runner implementations are external commands, not fbt core
packages. Official maintained adapters should live under `adapters/`, while
examples remain under `examples/`.

## 3. Package Naming

Recommended package names:

| Package | Provides | Command |
|---|---|---|
| `fbt-runner-openai` | `openai.responses` | `fbt-runner-openai` |
| `fbt-runner-anthropic` | `anthropic.messages` | `fbt-runner-anthropic` |
| `fbt-runner-gemini` | `gemini.generate_content` | `fbt-runner-gemini` |
| `fbt-runner-codex-cli` | `codex.cli` | `fbt-runner-codex-cli` |
| `fbt-runner-claude-code` | `claude_code.cli` | `fbt-runner-claude-code` |
| `fbt-runner-ollama` | `ollama.generate` | `fbt-runner-ollama` |

The package name identifies the integration family. The logical runner name
identifies the runner contract used by transforms. One package may provide
multiple logical runners when they share the same executable and dependency
set.

The source tree includes `adapters/openai` as the official OpenAI adapter used
by the practical examples. It is outside `internal/`, is invoked as an external
process, and reads `OPENAI_API_KEY` from the environment. The official Codex
CLI and Claude Code adapters are also outside core: they translate fbt policy
into the corresponding CLI flags where possible and fail before invoking the
CLI when the requested policy cannot be represented safely. Separately packaged
or project-local adapters should follow the same protocol and credential
boundary.

## 4. Distribution

Installation is out-of-band in MVP. Acceptable distribution paths include:

- `brew install ...`
- `go install ...`
- `npm install -g ...`
- `uv tool install ...`
- `pipx install ...`
- a checked-in project-local `plugins/<name>/` directory
- an internal package manager or artifact registry

fbt core must not download packages, vendor provider SDKs, or mutate
`fs_project.yml` as part of installation in MVP.

Official Go adapter modules must be installable from a clean environment without
the repository's `go.work` file or local `replace` directives. Source users can
install a current adapter command directly from the VCS module path:

```sh
go install github.com/nyuta01/fbt/adapters/openai/cmd/fbt-runner-openai@main
```

Release tags for nested modules must be cut per module path, for example
`adapters/openai/v0.1.0` for the OpenAI adapter and `sdk/go/v0.1.0` for the Go
runner SDK. The user-facing install command still uses the module version:

```sh
go install github.com/nyuta01/fbt/adapters/openai/cmd/fbt-runner-openai@v0.1.0
```

## 5. Project Config

Projects may reference an adapter directly:

```yaml
runners:
  - name: openai.responses
    type: llm
    protocol: stdio_jsonrpc
    command: fbt-runner-openai
    args: ["responses"]
    env:
      - OPENAI_API_KEY
    config:
      provider: openai
      default_model: gpt-5
```

CLI-agent adapters use the same shape:

```yaml
runners:
  - name: codex.cli
    type: agent
    protocol: stdio_jsonrpc
    command: fbt-runner-codex-cli
    args: ["--profile", "fbt"]
    env:
      - OPENAI_API_KEY
    config:
      agent: codex
      staging: isolated_worktree
```

Only environment variable names are stored. Values stay in the user's shell,
secret manager, CI environment, or adapter-specific credential mechanism.

## 6. Plugin Manifest

Adapter packages may include `fbt_plugin.yml` so fbt can discover them from
project or user plugin directories.

```yaml
name: fbt-runner-openai
version: 0.1.0
protocol: stdio_jsonrpc
command: bin/fbt-runner-openai
args: ["responses"]
provides:
  - runner: openai.responses
    type: llm
    transform_types: ["llm"]
    artifact_types: ["markdown", "markdown_directory", "text", "directory"]
env:
  - OPENAI_API_KEY
checksum:
  command: sha256:...
```

Manifest capabilities are advisory. The runner process must still return
authoritative capabilities from `initialize`.

## 7. PATH Convention

When a project omits explicit runner config and no plugin manifest is found,
fbt may resolve a logical runner through the PATH convention:

```text
runner: openai.responses
-> fbt-runner-openai-responses
```

Package-level commands such as `fbt-runner-openai` are still preferred when the
adapter needs subcommands, shared configuration, or multiple logical runners.
In that case, use project config or a plugin manifest.

## 8. CLI-Agent Adapter Requirements

Packages such as `fbt-runner-codex-cli` and `fbt-runner-claude-code` wrap an
external agent CLI. They must follow the safe adapter contract:

- the adapter process speaks fbt JSON-RPC; the external CLI does not need to
- the agent runs in a staging workspace, isolated copy, or scoped work tree
- fbt policy is mapped to the agent's permission, sandbox, network, tool,
  timeout, and turn controls where available
- execution fails closed when policy cannot be enforced safely
- final files are copied under `work.outputs`
- structured events and tool-call payloads are redacted
- official artifact paths and `.fbt/state` are not modified directly

See [Runner Protocol Spec](runner-protocol-spec.md) and
[Security and Conformance Spec](security-and-conformance-spec.md).

Adapters recommended for agent CLIs should pass the stricter safety profile:

```sh
python3 tests/runner-conformance/run.py \
  --runner-command 'fbt-runner-codex-cli' \
  --strict \
  --agent-adapter
```

The `--agent-adapter` profile checks that protocol messages are redacted, the
temporary source/logical artifact/state guard files are unchanged, the adapter
reports a staging workspace under `work.root` but outside `work.outputs`, and
policy mapping is explicitly fail-closed.

Use `--expect-policy-failure` with `--agent-adapter` to verify the negative
case: the harness sends an unsupported policy, and the adapter must return a
structured policy error before invoking the external CLI.

Official adapter policy mapping is intentionally conservative:

- Codex CLI adapter: maps fbt's read-only staging expectation to `codex exec
  --sandbox read-only`, applies fbt timeout as a process timeout, and fails
  closed for denied network, tool allow/deny lists, max tool calls, or max cost
  because those controls are not represented by the wrapped CLI invocation.
- Claude Code adapter: maps read/write/search/shell tool allow and deny lists
  to Claude Code tool flags, applies fbt timeout as a process timeout, maps
  `max_cost_usd` to `--max-budget-usd`, and fails closed for denied network,
  unknown tools, or max tool calls.

Official CLI-agent adapter tests use small executable fixtures under
`adapters/*/testdata/` so `make verify` can check protocol and safety behavior
without network, credentials, or paid model calls. Those fixtures are not
user-facing demos; normal projects invoke `codex exec` or `claude -p` through
the adapter.

## 9. Versioning

Adapter packages should use semantic versioning. Breaking changes include:

- dropping a supported fbt protocol version
- removing a logical runner name
- removing transform or artifact type support
- changing required environment variable names
- changing output-candidate semantics

Recommended release metadata:

- package version
- supported fbt protocol versions
- supported transform types and artifact types
- provider API or CLI version tested by the adapter
- credential environment variable names
- checksum or signature for distributed binaries when available

## 10. Opt-In Smoke Matrix

`make verify` stays deterministic and service-free. To validate installed
provider or CLI-agent adapters on a workstation or CI secret context, use:

```sh
FBT_RUNNER_ADAPTER_SMOKE_MATRIX='openai.responses|llm|markdown|fbt-runner-openai responses|OPENAI_API_KEY|false
codex.cli|agent|markdown|fbt-runner-codex-cli --profile fbt|OPENAI_API_KEY|true
claude_code.cli|agent|markdown|fbt-runner-claude-code --profile fbt|ANTHROPIC_API_KEY|true
gemini.generate_content|llm|markdown|fbt-runner-gemini generate-content|GEMINI_API_KEY|false
internal.agent|agent|markdown|company-fbt-agent --mode fbt|INTERNAL_AGENT_TOKEN|true' \
make runner-adapter-smoke
```

Each row is:

```text
logical_name|runner_type|artifact_type|command|required_env_csv|agent_adapter
```

The smoke script validates each row by:

- running runner conformance against the command
- adding `--agent-adapter` for CLI-agent rows
- generating a temporary fbt project with that logical runner
- running `fbt doctor`
- running `fbt plan --select adapter_smoke`

Set `FBT_RUNNER_ADAPTER_SMOKE_BUILD=1` to also run a real build and inspect the
committed artifact. Keep this explicit because it may call paid providers or
local agents. Set `FBT_RUNNER_ADAPTER_SMOKE_TIMEOUT_SECONDS` when an installed
adapter needs a longer conformance timeout.

## 11. Conformance Checklist

Before documenting an adapter as fbt-compatible:

1. Run `FBT_RUNNER_CONFORMANCE_COMMAND='adapter-command' make runner-conformance`.
2. Create a temporary fbt project using the adapter.
3. Run `fbt doctor`.
4. Run `fbt plan` and confirm expected dirty/block reasons.
5. Run a build that writes at least one output candidate.
6. Confirm no raw credentials or raw prompts are persisted in state,
   OpenLineage, or OTel exports.
7. Confirm failed or denied runs do not update official artifact paths.

For CLI-agent adapters, also run `tests/runner-conformance/run.py` with
`--strict --agent-adapter` for the supported policy path and with
`--strict --agent-adapter --expect-policy-failure` for an unsupported policy
path before recommending the package.

Adapters that require real provider accounts should keep provider smoke tests
behind explicit opt-in commands such as `make runner-adapter-smoke`, not
`make verify`.

## 12. Non-Goals

Adapter packaging does not add:

- provider SDKs to `fbt` core
- in-process plugin loading
- a package marketplace in MVP
- direct credential storage in fbt state
- provider-specific subcommands in the base CLI
- a custom visualization backend

Optional adapters extend fbt by satisfying the same runner protocol and
discovery rules as any other external process.
