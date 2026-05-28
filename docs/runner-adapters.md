# fbt Runner Adapter Packaging

Status: Draft  
Created: 2026-05-28  
Audience: maintainers and authors of optional provider or agent integrations

## 1. Scope

`fbt` core stays provider-free and runtime-free. OpenAI, Anthropic, Gemini,
Codex CLI, Claude Code, local model servers, document converters, SaaS
connectors, and internal agents belong in optional out-of-process runner
packages.

An adapter package is responsible for:

- shipping one or more executable runner commands
- owning provider SDK, agent runtime, and converter dependencies
- reading credentials from its own environment
- speaking the fbt stdio JSON-RPC runner protocol
- advertising runtime capabilities through `initialize`
- writing only output candidates under `work.outputs`
- documenting its own provider-specific setup and limits

fbt core is responsible only for discovery, invocation, capability validation,
state, policy/eval/review checks, descriptors, and official commits.

## 2. Package Naming

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

## 3. Distribution

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

## 4. Project Config

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

## 5. Plugin Manifest

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

## 6. PATH Convention

When a project omits explicit runner config and no plugin manifest is found,
fbt may resolve a logical runner through the PATH convention:

```text
runner: openai.responses
-> fbt-runner-openai-responses
```

Package-level commands such as `fbt-runner-openai` are still preferred when the
adapter needs subcommands, shared configuration, or multiple logical runners.
In that case, use project config or a plugin manifest.

## 7. CLI-Agent Adapter Requirements

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

## 8. Versioning

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

## 9. Conformance Checklist

Before documenting an adapter as fbt-compatible:

1. Run `FBT_RUNNER_CONFORMANCE_COMMAND='adapter-command' make runner-conformance`.
2. Create a temporary fbt project using the adapter.
3. Run `fbt doctor`.
4. Run `fbt runner validate RUNNER_NAME`.
5. Run `fbt plan` and confirm expected dirty/block reasons.
6. Run a build that writes at least one output candidate.
7. Confirm no raw credentials or raw prompts are persisted in state, docs,
   OpenLineage, or OTel exports.
8. Confirm failed or denied runs do not update official artifact paths.

Adapters that require real provider accounts should keep provider smoke tests
behind explicit opt-in commands, not `make verify`.

## 10. Non-Goals

Adapter packaging does not add:

- provider SDKs to `fbt` core
- in-process plugin loading
- a package marketplace in MVP
- direct credential storage in fbt state
- provider-specific subcommands in the base CLI
- a custom visualization backend

Optional adapters extend fbt by satisfying the same runner protocol and
discovery rules as any other external process.
