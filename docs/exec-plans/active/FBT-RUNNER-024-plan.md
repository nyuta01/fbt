# FBT-RUNNER-024 Fail visibly on CLI-agent staged input truncation

## Observation

Codex CLI and Claude Code adapters stage input files through `io.LimitReader`
with a 2 MiB cap. The adapter does not currently report whether a file was
truncated, so the external agent may generate from incomplete evidence.

This is worse than a visible failure: the generated artifact can look valid
while being based on incomplete source or prompt context.

## Decision

Do not silently truncate staged evidence. Either stage the full file within an
explicit configured limit or fail before invoking the external CLI with a clear
error naming the oversized file and limit.

Keep the current 2 MiB per-file staging limit as an explicit adapter boundary.
Copy files fully when they are within the limit, and reject source or asset
files above the limit before the external CLI starts.

## Permanent Fix

Add adapter tests with oversized source and asset files. The tests should prove
that no external CLI is invoked after an oversized staging failure and that the
error is actionable.

## Next Check

Done. Codex CLI and Claude Code adapter tests cover oversized source and asset
files, agent-adapter conformance passes, and `make verify` passes.
