# FBT-STD-003 Add OpenTelemetry-compatible execution telemetry export

## Observation

fbt records invocation and transform summaries in `run_results.jsonl`, but
execution telemetry was not exportable in a standard backend-compatible shape.
Runner usage and tool-call events existed at the protocol boundary, yet only
summary usage/provenance was persisted for builds.

## Decision

Implement `fbt export otel [--output PATH]` as a local-first OTLP/JSON trace
payload export. Build invocations become root spans, transform runs become
child spans, usage/model/cost fields become span attributes, and safe runner
events become span events. Do not add a network exporter or backend dependency
to the base CLI.

## Permanent Fix

Persist transform run start/end timestamps and redacted runner events in
`run_results.jsonl`. Added an `internal/telemetry` OTLP/JSON exporter and wired
it into `fbt export otel`, with Go and CLI smoke coverage.

## Next Check

Run:

```sh
make verify
```
