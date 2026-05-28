# FBT-RUNNER-004 Define safe CLI-agent adapter contract

## Observation

fbt can discover and invoke arbitrary external runner commands, but existing
agent CLIs such as Codex CLI or Claude Code have their own permission and output
models. Without an explicit adapter contract, a wrapper author could let an
agent write directly to managed artifacts or emit unredacted tool data.

## Decision

Treat the adapter process as the fbt runner. External agent CLIs remain behind
that adapter, run in a staging workspace, and return only redacted events plus
output candidates under `work.outputs`. Core keeps the official commit boundary
and validates containment even when an adapter misbehaves.

## Permanent Fix

Documented the CLI-agent adapter contract in the runner protocol, security
spec, and usage guide. Added a fake-runner escape mode plus Go and conformance
coverage proving output candidates outside `work.outputs` are rejected before
official artifact commit.

## Next Check

Run:

```sh
make verify
```
