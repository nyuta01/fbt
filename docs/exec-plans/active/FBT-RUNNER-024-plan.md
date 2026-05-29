# FBT-RUNNER-024 Fail visibly on CLI-agent staged input truncation

## Observation

Codex CLI and Claude Code adapters stage input files through `io.LimitReader`
with a 2 MiB cap. The adapter does not currently report whether a file was
truncated, so the external agent may generate from incomplete evidence.

## Decision

Do not silently truncate staged evidence. Either stage the full file within an
explicit configured limit or fail before invoking the external CLI with a clear
error naming the oversized file and limit.

## Permanent Fix

Add adapter tests with oversized source and asset files. The tests should prove
that no external CLI is invoked after an oversized staging failure and that the
error is actionable.

## Next Check

Run Codex CLI and Claude Code adapter tests, agent-adapter conformance, then
`make verify`.
