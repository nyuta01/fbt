# Harness Engineering

This repository follows an AI-first operating model. The harness is the
repository-local environment that lets coding agents do reliable work: compact
instructions, source-of-truth docs, structured task state, verification scripts,
failure logs, and handoff memory.

The goal is not to create heavy process. The goal is to make the next agent run
restartable, bounded, and mechanically verifiable.

## Principles

### Small Agent Entry Point

`AGENTS.md` must stay short. It routes agents to the current source of truth
instead of duplicating the design.

Primary references:

- `README.md`
- `docs/design-doc.md`
- `docs/spec.md`
- `docs/project-config-spec.md`
- `docs/runner-protocol-spec.md`
- `docs/methodology/self-pdca-loop.md`
- `docs/methodology/permanent-fix-protocol.md`
- `docs/QUALITY_SCORE.md`
- `docs/exec-plans/feature-list.json`
- `docs/exec-plans/active/`
- `docs/agent-failures.md`
- `AGENT_PROGRESS.md`

### Repository As System Of Record

Decisions that matter to future agents belong in the repository. Do not rely on
chat history, memory, or external notes for product behavior, architecture
constraints, or verification expectations.

### Structured Task State

Task state lives in `docs/exec-plans/feature-list.json`. Each task has a stable
id, priority, status, affected paths, dependencies, plan URL, verification
gates, and notes.

### One Task At A Time

Agents should pick one highest-priority non-done task, complete it, verify it,
and update task state before expanding scope.

### Shift Feedback Left

`make verify` is the single repository gate. It starts with harness, drift, and
docs checks. Product checks live behind the same target.

### CI Runs The Same Gate

GitHub Actions runs `make verify` on pull requests and pushes to `main`. CI must
not grow a second checklist of product checks; it prepares the toolchain and
invokes the same Make target agents run locally.

### Permanent Fixes

If a mistake repeats, do not rely on "be careful next time." Record it in
`docs/agent-failures.md` and prevent recurrence with the smallest reliable
guard: a script, test, spec update, or concise AGENTS rule.

### Self-PDCA

Every non-trivial task follows the loop in
`docs/methodology/self-pdca-loop.md`: plan one bounded task, make the change,
check it with deterministic evidence, then act by updating a guardrail, quality
score, task state, or failure log.

### Local-First Is A Harness Constraint

`fbt` must remain usable without a daemon, scheduler, metadata DB, web server,
or cloud account. The harness should reject drift that makes the base tool
heavy or hides required services behind the verification path.

## Current Verification Gates

| Gate | Command | Purpose |
|---|---|---|
| `ci_verify` | GitHub Actions `make verify` | Enforces the repository gate on pull requests and `main` pushes |
| `harness_check` | `make harness-check` | Validates required harness files and feature-list shape |
| `drift_check` | `make drift-check` | Validates active plan references and failure-log status |
| `validate_docs` | `make validate-docs` | Validates docs-local links and English-only docs |
| `fmt_check` | `make fmt-check` | Verifies Go formatting |
| `go_test` | `make go-test` | Runs Go unit tests |
| `cli_smoke` | `make cli-smoke` | Exercises the minimal CLI scaffold |

## Target Harness Growth

As product code lands, `make verify` should add deterministic checks for:

- Project config parsing.
- Manifest generation.
- Runner protocol validation.
- State and run result writes.
- CLI command behavior.
- Local-first smoke scenarios.
- Security-sensitive filesystem boundaries.

Each product behavior change must be anchored in a spec, design doc, or active
plan before it is considered done.

