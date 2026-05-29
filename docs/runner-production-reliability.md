# Runner Production Reliability

Status: MVP-ready
Audience: official adapter maintainers and production runner authors

## Purpose

fbt core stays provider-free. Production reliability for OpenAI, Claude Code,
Codex, Gemini, converters, and internal tools belongs in external runners.
This document defines the minimum production contract that those runners should
meet before a team treats generated artifacts as operational evidence.

## Contract

| Concern | Runner requirement | fbt expectation |
|---|---|---|
| Input-size failure | Refuse oversized source or asset files before invoking the provider or agent. | Failed receipts remain inspectable and current artifact pointers do not advance. |
| Retry classification | Return structured errors that distinguish retryable provider/runtime failures from policy or contract failures. | Users can retry explicitly with `build --failed`; fbt does not run an automatic retry queue. |
| Provider metadata | Report runner name, runner version, provider, model, token usage, cost where available, and elapsed time. | Receipts and standard exports can explain which external runtime produced the artifact. |
| Redaction | Never emit raw credentials, secret env values, or sensitive source content in protocol events, stderr, or errors. | Core stores bounded, redacted diagnostics and rejects leaked redaction markers in conformance. |
| Timeout mapping | Map fbt policy timeouts to provider request timeouts or agent process timeouts. | Timeout failures are visible failed-run receipts, not partial artifact commits. |
| Fail-closed policy | Refuse execution when a requested network, tool, cost, or sandbox policy cannot be represented safely. | Core records a failed run and does not promote output candidates. |
| Output boundary | Write candidates only under `work.outputs` and declare them through `fbt/outputCandidate`. | Core recomputes descriptors and commits only contained candidates. |
| Live opt-in | Keep live provider/agent calls behind explicit credentials, installed commands, or opt-in env vars. | `make verify` remains deterministic and provider-free by default. |

## Official Adapter Baseline

The repository-maintained adapters are expected to satisfy the contract as
follows:

| Adapter | Baseline |
|---|---|
| `fbt-runner-openai` | Reads `OPENAI_API_KEY`, reports provider/model/usage/cost metadata, supports fake-response conformance, redacts configured secrets, and keeps live calls opt-in. |
| `fbt-runner-codex-cli` | Stages inputs before invoking Codex CLI, enforces a per-file staging limit, maps timeout/sandbox policy, and fails closed for unsupported network/tool/cost controls. |
| `fbt-runner-claude-code` | Stages inputs before invoking Claude Code, enforces a per-file staging limit, maps timeout/tool/budget controls where supported, and fails closed for unsupported policy. |
| `fbt-runner-command` | Executes declared command transforms and reports files written under `work.outputs`; command authors own domain retry behavior. |

## Verification Ladder

Use these checks before live production rollout:

```sh
make official-adapter-smoke
make production-pilot-smoke
```

Then run opt-in live checks only in an environment that intentionally provides
credentials and authenticated CLIs:

```sh
OPENAI_API_KEY=... FBT_PILOT_LIVE_OPENAI=1 scripts/pilot-daily-real-adapters.sh
```

The live check should capture latency, cost, model, and output-quality notes in
`docs/pilots/` or the team's own runbook. Do not add live provider calls to
`make verify`.
