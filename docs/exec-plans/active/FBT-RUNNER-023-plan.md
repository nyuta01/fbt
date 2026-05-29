# FBT-RUNNER-023 Remove JSON-RPC JSONL scanner size limits

## Observation

The core protocol client and Go SDK stdio server use `bufio.Scanner` without
raising the default token limit. Larger JSON-RPC JSONL messages can fail before
the runner or core sees a structured protocol error.

The failure mode is especially likely for many-file requests or structured
progress notifications that cross Go's default scanner token size even though
they are still reasonable protocol frames.

## Decision

Make the JSONL message limit explicit and large enough for many-file fbt
requests and structured notifications, or replace scanner usage with a reader
that returns complete lines up to a controlled maximum.

The current implementation keeps JSONL framing and sets a 16 MiB explicit frame
limit in both the core client and Go SDK stdio server. Raw source, prompt, and
artifact bodies still belong in files rather than protocol messages.

## Permanent Fix

Add tests for large initialize/run/event messages in both the core client and
SDK server. Document the message-size boundary in the runner protocol spec if a
limit remains.

## Next Check

Done. Core protocol and SDK stdio tests cover messages above Go's default
scanner limit, runner conformance passes, and `make verify` passes.
