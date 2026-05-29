# FBT-RUNNER-023 Remove JSON-RPC JSONL scanner size limits

## Observation

The core protocol client and Go SDK stdio server use `bufio.Scanner` without
raising the default token limit. Larger JSON-RPC JSONL messages can fail before
the runner or core sees a structured protocol error.

## Decision

Make the JSONL message limit explicit and large enough for many-file fbt
requests and structured notifications, or replace scanner usage with a reader
that returns complete lines up to a controlled maximum.

## Permanent Fix

Add tests for large initialize/run/event messages in both the core client and
SDK server. Document the message-size boundary in the runner protocol spec if a
limit remains.

## Next Check

Run protocol and SDK tests, runner conformance, then `make verify`.
