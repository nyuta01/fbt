# FBT-MVP-013 Implement AI-native runner experience

## Observation

The runner protocol and build lifecycle support external protocol runners, but
the repository only has fake and command runners. LLM and agent transform
examples need protocol-compatible runners with usage and tool-call reporting
without adding provider SDKs to the base runtime.

## Decision

Implement optional local AI runner examples:

- add `runners/llm` for deterministic mock LLM output, usage, cost, and
  provenance reporting
- add `runners/agent` for deterministic mock agent output, usage, provenance,
  and redacted tool-call events
- pass transform model/tools from build into `fbt/runTransform`
- record runner usage/provenance in transform run records
- document the examples as replaceable out-of-process runners

## Permanent Fix

Added protocol tests for both optional AI runners and a build lifecycle test
that runs the local LLM runner, verifies model metadata propagation, and checks
usage/provenance in run results.

## Next Check

Run:

```sh
make verify
```

Result on 2026-05-28: pass.
