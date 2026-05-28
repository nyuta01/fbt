# FBT-AI-001 Add opt-in real LLM runner smoke

## Observation

The repository has deterministic local LLM and agent runners for default
verification, but no safe way to smoke-test an externally installed real LLM
runner without modifying `make verify` or adding provider SDKs to fbt core.

## Decision

Add an environment-gated smoke target. It only runs a real external runner when
`FBT_REAL_LLM_RUNNER_COMMAND` is set, otherwise it reports `skipped` and exits
successfully. Provider credentials, SDKs, network access, and billing remain
outside fbt core and outside the default verification gate.

## Permanent Fix

Added `scripts/smoke-real-llm.sh` and `make real-llm-smoke`. The script creates
a temporary project, wraps the external runner command, runs `fbt doctor`,
builds one LLM transform, and checks the committed artifact. Usage docs and the
runner protocol spec now explain the opt-in contract.

## Next Check

Run:

```sh
make verify
make real-llm-smoke
```

The second command is expected to skip unless `FBT_REAL_LLM_RUNNER_COMMAND` is
set.
