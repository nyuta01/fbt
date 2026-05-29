# Official Runner Adapter Design Report

Created: 2026-05-29
Audience: fbt maintainers, runner authors, and future official adapter owners

## 1. Conclusion

Official fbt runners should be maintained as external adapter packages, not as
implementation code inside fbt core.

The recommended product shape is:

```text
fbt core
  owns: project graph, planning, build receipts, output commit, evals, lineage

official adapter packages
  own: provider SDKs, CLI-agent invocation, converter logic, credentials,
        packaging, release notes, live smoke tests, and provider-specific docs
```

This matches the direction already documented in fbt: the runner is an external
command that speaks the fbt runner protocol. The important change is operational:
official adapters must be treated as first-class maintained packages, with
versioning, checksums, conformance, security policy, docs, and release cadence.

The first official packages should be:

1. `fbt-runner-command`: a generic command adapter for Unix-style composition.
2. `fbt-runner-openai`: a direct OpenAI Responses API adapter.
3. `fbt-runner-codex-cli`: a safe wrapper around `codex exec`.
4. `fbt-runner-claude-code`: a safe wrapper around `claude -p`.

`fbt-runner-anthropic`, `fbt-runner-gemini`, and local-model adapters should
follow once the first two packages prove the release and support model.

## 2. Current fbt Baseline

The current core boundary is sound:

- runners are external commands
- the transport is JSON-RPC 2.0 compatible messages over stdio
- capability negotiation happens through `initialize`
- core computes official artifact descriptors and commits output candidates
- provider SDKs and agent runtimes are outside core
- source-checkout examples live under `examples/runner_adapters/`
- test-only protocol fixtures live under `tests/runner_fixtures/`
- installed adapter validation is opt-in through `make runner-adapter-smoke`

The gap is not conceptual. The gap is that fbt has example adapters but no
officially supported adapter package with a clear maintenance contract.

## 3. External Reference Findings

| Reference | Relevant practice | fbt implication |
|---|---|---|
| Go `plugin` package | Go's in-process plugin mechanism has portability, race detector, initialization, security, toolchain-version, and dependency-source drawbacks. The Go docs explicitly point many designs toward IPC/RPC instead. | Do not use Go shared-library plugins for fbt runners. |
| HashiCorp `go-plugin`, Terraform, Packer, Vault | Plugins are standalone processes that communicate with core over RPC. Provider/plugin packages are versioned and often verified by checksums. Vault also emphasizes process isolation and plugin catalog integrity. | fbt's out-of-process runner protocol is aligned with mature Go infrastructure tooling. Add package integrity and version discipline before adding install UX. |
| Terraform providers | Provider requirements separate local names, source addresses, and version constraints. Dependency lock files pin selected provider versions and checksums. | Future fbt plugin install/lock should pin adapter source, version, protocol compatibility, and checksums. MVP can keep installation out-of-band. |
| Packer plugins | A plugin is a binary with naming conventions, semantic versions, API versions, OS/arch-specific artifacts, and matching SHA256 files. | Official fbt adapters should ship platform binaries and checksum files even before fbt has a package manager. |
| Git and kubectl | Extensions are ordinary executables on `PATH` with naming conventions (`git-*`, `kubectl-*`). They keep the core small and allow any implementation language. | Keep the `fbt-runner-*` PATH convention. Do not require adapters to be written in Go. |
| dbt adapters | dbt distinguishes adapter ecosystem quality through "trusted" criteria: feature coverage, tests, documentation, release cadence, community responsiveness, and security practices. | fbt should define "official adapter" as a quality and maintenance tier, not merely "code lives nearby." |
| MCP | MCP validates JSON-RPC over stdio as a common local AI-tool integration pattern. It also shows that process execution and stdio transports require explicit safety boundaries. | fbt's stdio JSON-RPC choice is reasonable, but agent adapters must be fail-closed and heavily tested. |
| Codex CLI and Claude Code | Both expose non-interactive automation modes with structured output and explicit permission/sandbox controls. | fbt should wrap these CLIs through adapters rather than reimplement agent loops. |
| Cobra and pflag | Cobra encourages hierarchical commands, local flags by default, sparse persistent flags, validation hooks, and clear help. | Adapter CLIs may use Cobra for `serve`, `doctor`, and `version`, but the runner protocol path should stay simple and scriptable. |
| Go modules | `go install module@version` installs executable packages without mutating the current module, and major versions use module path suffixes. | Official Go adapters should be installable with `go install ...@version` and should not add provider SDK dependencies to fbt core. |

## 4. Design Decision

### Adopt External Official Adapter Packages

Official adapters should live in separate repositories or separate Go modules
that do not participate in fbt core's `go test ./...` dependency graph.

Preferred repository shape:

```text
github.com/nyuta01/fbt-runner-openai
  cmd/fbt-runner-openai/
  internal/openaiadapter/
  internal/fbtprotocol/
  testdata/
  fbt_plugin.yml
  README.md
  CHANGELOG.md
  SECURITY.md
```

If the project temporarily keeps official adapters in the same Git repository,
each adapter must still be a nested module with its own `go.mod`, dependency
tree, CI, release artifacts, and docs. It should not appear as a top-level
`runners/` directory in fbt core.

### Add a Tiny Public Runner SDK

Official adapters need shared protocol types and helper code, but they must not
import fbt `internal/` packages.

Recommended follow-up:

```text
github.com/nyuta01/fbt-runner-sdk-go
  protocol types
  JSON-RPC JSONL stdio server helpers
  output-candidate helpers
  redaction helpers
  test harness helpers
```

This package must stay provider-free. It is not a plugin runtime and must not
implement transform logic. It only lowers the cost of writing conformant
external commands.

An alternative is `github.com/nyuta01/fbt/pkg/runnerprotocol`, but a separate
module better preserves the provider-free core dependency graph and release
independence.

### Defer Package Installation in Core

Keep MVP installation out-of-band:

```sh
go install github.com/nyuta01/fbt-runner-openai/cmd/fbt-runner-openai@v0.1.0
```

or:

```sh
brew install nyuta01/tap/fbt-runner-openai
```

`fbt plugin install` should remain reserved until fbt can verify source,
version, checksum/signature, OS/arch compatibility, and manifest contents
without turning core into a package manager.

## 5. Official Adapter Contract

An official adapter package must provide:

- an executable command named `fbt-runner-<integration>`
- a `fbt_plugin.yml` manifest
- `--help` and `--version`
- protocol support for `initialize`, `fbt/validate`, and `fbt/runTransform`
- capability declarations for transform and artifact types
- strict output-candidate writing under `work.outputs`
- no direct writes to artifact destinations or `.fbt/state`
- redacted logs and protocol events
- documented credential environment variables
- deterministic unit tests
- strict runner conformance in CI
- optional live smoke tests gated by credentials
- release artifacts with checksums
- a compatibility table for fbt protocol versions

Official packages should use semantic versioning. Breaking changes include:

- dropping a supported fbt protocol version
- removing a logical runner name
- removing transform or artifact type support
- changing required credential names
- changing output-candidate semantics
- weakening policy enforcement

## 6. Runner-Specific Recommendations

### `fbt-runner-command`

Purpose: wrap existing Unix tools and project-local scripts while keeping fbt
responsible for graph/state/artifact receipts.

Why first:

- no provider dependency
- proves that official adapters are not only LLM providers
- strengthens the "one tool that composes with other tools" position
- gives users a practical bridge to remark, Pandoc, dbt artifacts, DataChain
  outputs, internal scripts, and converters

Implementation notes:

- accept command configuration only from fbt runner request/config, not from
  source file contents
- run in a scoped working directory
- map fbt policy to allowed executable, args, cwd, env, timeout, output paths,
  and size limits
- require declared output candidates
- stream command stderr as redacted runner events
- record exit code and timing in runner metadata

### `fbt-runner-openai`

Purpose: produce text/Markdown artifacts through the OpenAI Responses API.

Why second:

- it covers the main LLM use case without agent filesystem side effects
- official OpenAI Go SDK support exists
- credentials and costs are simpler than CLI-agent wrappers

Implementation notes:

- use the official OpenAI Go SDK in the adapter package, not in fbt core
- read credentials from `OPENAI_API_KEY` or a documented adapter-specific
  mechanism
- default to a conservative text/Markdown generation path
- support explicit model configuration from the fbt transform model/config
- emit usage and cost metadata when available
- never print prompts, sources, API keys, or full provider responses to stderr
  unless an explicit debug flag is set and redaction is applied
- provide live smoke tests only behind `OPENAI_API_KEY`

### `fbt-runner-codex-cli`

Purpose: adapt Codex CLI's non-interactive automation mode to the fbt runner
protocol.

Implementation notes:

- invoke `codex exec` inside an isolated staging workspace
- use explicit sandbox and approval settings
- prefer JSON/JSONL output for machine parsing
- use `CODEX_API_KEY` only for the single invocation when key auth is used
- avoid loading uncontrolled user config in deterministic CI flows
- copy final output candidates into `work.outputs`
- fail closed if policy cannot be mapped to Codex permissions
- pass the strict `--agent-adapter` conformance profile

### `fbt-runner-claude-code`

Purpose: adapt Claude Code headless mode to the fbt runner protocol.

Implementation notes:

- invoke `claude -p` in non-interactive mode
- prefer `--bare` for scripted deterministic calls
- use `--output-format json` or `stream-json` where possible
- map policy to allowed tools, permission mode, max turns, budget, cwd, and
  environment
- read credentials from `ANTHROPIC_API_KEY` or documented Claude Code auth
  setup
- copy final output candidates into `work.outputs`
- fail closed if policy cannot be mapped safely
- pass the strict `--agent-adapter` conformance profile

## 7. Implementation Plan

### Phase 1: Official Adapter Foundation

Deliverables:

- `fbt-runner-sdk-go` or equivalent public runner protocol module
- adapter repository template
- official adapter acceptance checklist
- CI workflow that runs:
  - Go formatting/tests
  - runner conformance
  - agent-adapter conformance where applicable
  - docs link checks
  - release dry-run
- documentation that distinguishes:
  - example adapter
  - official adapter
  - third-party adapter
  - project-local adapter

### Phase 2: `fbt-runner-command`

Deliverables:

- official command adapter package
- real examples showing Markdown, Pandoc-style, and internal-script use
- migration from source-checkout command example to official install docs
- opt-in smoke row for `make runner-adapter-smoke`

### Phase 3: `fbt-runner-openai`

Deliverables:

- official OpenAI adapter package
- incident runbook and support manual examples updated to use the installed
  package path
- live smoke workflow gated by `OPENAI_API_KEY`
- usage/cost metadata tests with mocked provider responses

### Phase 4: CLI-Agent Adapters

Deliverables:

- `fbt-runner-codex-cli`
- `fbt-runner-claude-code`
- staging-workspace implementation
- policy mapping tests
- redaction tests
- real examples with bounded prompts and visible output candidates

### Phase 5: Install and Lock UX

Deliverables:

- `fbt plugin install` only after checksum/signature and lock semantics are
  specified
- future `fbt.lock.json` records adapter source, version, checksum, OS/arch,
  protocol version, and logical runners
- `fbt doctor` explains stale, missing, or incompatible installed adapters

## 8. Operating Model

Official adapters need a higher bar than examples:

| Area | Minimum bar |
|---|---|
| Ownership | Named maintainer and security contact |
| Release cadence | Patch releases for provider/API breakage; regular dependency updates |
| Compatibility | Explicit fbt protocol compatibility table |
| Tests | Deterministic unit tests, conformance, and opt-in live smoke |
| Security | No secret logging; fail-closed policy; dependency scanning |
| Supply chain | Checksums for release artifacts; signed releases when available |
| Docs | Install, credentials, project config, examples, troubleshooting |
| Deprecation | Announced support window and migration path |

Borrow dbt's "trusted adapter" idea: "official" is a maintenance promise, not a
filesystem location.

## 9. Changes Needed in fbt Core

Core does not need provider logic. The core-side improvements are small:

1. Keep the runner protocol stable and schema-backed.
2. Add a public, provider-free runner SDK or protocol module for adapter
   authors.
3. Add an official adapter index document.
4. Keep `make verify` service-free.
5. Keep live provider/agent checks behind opt-in environment variables.
6. Add lock/install semantics only after checksum and source-address design is
   complete.

Do not add:

- provider SDKs to fbt core
- agent runtimes to fbt core
- in-process Go plugins
- a daemon or scheduler
- a metadata database
- an auto-downloading plugin manager without lock/checksum semantics

## 10. Sources Consulted

- Go `plugin` package: https://go.dev/pkg/plugin/
- Go modules reference: https://go.dev/ref/mod/
- Cobra command and flag guides: https://cobra.dev/docs/how-to-guides/working-with-commands/ and https://cobra.dev/docs/how-to-guides/working-with-flags/
- HashiCorp go-plugin: https://github.com/hashicorp/go-plugin
- Terraform plugin development: https://developer.hashicorp.com/terraform/plugin
- Terraform provider requirements: https://developer.hashicorp.com/terraform/language/providers/requirements
- Terraform provider locking: https://developer.hashicorp.com/terraform/cli/commands/providers/lock
- Packer plugin creation: https://developer.hashicorp.com/packer/docs/plugins/creation
- Packer plugin installation: https://developer.hashicorp.com/packer/docs/commands/plugins/install
- Packer plugin loading specification: https://developer.hashicorp.com/packer/docs/plugins/creation/plugin-load-spec
- Vault plugin architecture: https://developer.hashicorp.com/vault/docs/plugins/plugin-architecture
- Kubernetes kubectl plugins: https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/
- Git external command behavior: https://git-scm.com/docs/git
- dbt trusted adapters: https://docs.getdbt.com/docs/trusted-adapters
- dbt adapter creation guide: https://docs.getdbt.com/guides/adapter-creation
- JSON-RPC 2.0 specification: https://www.jsonrpc.org/specification
- Model Context Protocol transports: https://modelcontextprotocol.io/specification/2025-06-18/basic/transports
- OpenAI Codex non-interactive mode: https://developers.openai.com/codex/noninteractive
- Claude Code headless mode: https://code.claude.com/docs/en/headless
- OpenAI Go SDK: https://github.com/openai/openai-go
