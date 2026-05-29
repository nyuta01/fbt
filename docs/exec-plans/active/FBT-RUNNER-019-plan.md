# FBT-RUNNER-019 Fix CLI-agent live invocation issues

## Observation

Live Codex CLI adapter conformance failed even though `codex exec` worked
directly. The adapter forwarded the conformance model name `fixture` to Codex,
which the real CLI rejected. The first failure was also hard to diagnose
because adapter errors did not include bounded CLI output. Claude Code adapter
live conformance then revealed that variadic CLI flags consumed the prompt
argument before authentication was reached.

## Decision

Do not forward conformance fixture model names to real CLI agents. Capture
bounded combined CLI output in adapter errors so live failures are actionable.
Remove Claude Code variadic flags that swallowed the prompt; the adapter already
runs from the staging workspace.

## Permanent Fix

Codex CLI live conformance now reaches the real CLI and passes with saved Codex
authentication. Claude Code live conformance now reaches real authentication
and fails clearly when the local CLI is not logged in.

## Next Check

Done. `make verify` passes. To finish live provider coverage, set
`OPENAI_API_KEY` for OpenAI and log in Claude Code or set `ANTHROPIC_API_KEY`
for `claude --bare`.
