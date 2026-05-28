# FBT-MVP-008 Implement JSON-RPC runner protocol client

## Observation

Runner discovery can find executable commands, but core cannot start a runner
process, negotiate protocol capabilities, send `fbt/runTransform`, collect
events/output candidates, or surface structured protocol errors.

## Decision

Implement a stdio JSONL JSON-RPC client:

- start runner processes with context cancellation
- send `initialize`, `initialized`, `fbt/runTransform`, and
  `$/cancelRequest`
- read JSONL responses and notifications
- collect `fbt/event` and `fbt/outputCandidate` notifications per request
- return structured protocol errors for JSON-RPC error responses and malformed
  messages
- add a runner package bridge from discovered runners to protocol clients

## Permanent Fix

Added protocol fake-runner tests for initialize, `initialized`,
`fbt/runTransform`, streamed events, output candidates, JSON-RPC error
responses, and context cancellation.

## Next Check

Run:

```sh
make verify
```

Result on 2026-05-28: pass.
