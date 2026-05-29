# Daily Real Adapter Pilot

Date: 2026-05-30
Task: `FBT-PROD-001`

## Purpose

Prove that the daily support knowledge loop can run with official
production-capable adapters instead of the deterministic demo LLM/agent
runners, while keeping live provider calls opt-in.

## Command

```sh
make production-pilot-smoke
```

The smoke copies `examples/daily_qa_ops` to a temporary project and rewires:

| Transform | Adapter used by the pilot | Default execution |
|---|---|---|
| `daily_qa_candidates` | official `fbt-runner-openai` adapter | fake OpenAI response, no network call |
| `promote_manual_update` | official `fbt-runner-codex-cli` adapter | fixture Codex CLI command |

For an intentional live OpenAI call:

```sh
OPENAI_API_KEY=... FBT_PILOT_LIVE_OPENAI=1 scripts/pilot-daily-real-adapters.sh
```

Codex CLI remains fixture-backed unless `FBT_CODEX_CLI_COMMAND` points at a
real authenticated `codex` command.

## Observations

- The same `ops/run-daily.sh` wrapper works with official adapters.
- `doctor` validates both protocol runners before `build`.
- The run bundle still contains source-window validation, plan/build output,
  artifact inspection, retention report, OpenLineage events, OTel traces, and
  quality gates.
- Live provider cost, latency, and output quality are intentionally not part of
  the default verification gate; they are opt-in pilot evidence.

## Decision

The production pilot path should stay as an external smoke and docs pattern.
fbt core remains provider-free and does not gain scheduler, retry queue, or
model-specific execution logic.

## Next Check

Use `FBT-RUNNER-027` to harden the production runner reliability contract based
on this pilot shape: failure classification, metadata, redaction, timeout, and
fail-closed policy behavior.
