# FBT-RUNNER-008 Add optional OpenAI Responses runner

## Observation

The practical manual-generation examples used real external runner
configuration, but a source checkout did not include an executable
provider-backed runner. Users with an OpenAI API key still needed to install an
adapter before they could run the examples end to end.

## Decision

Add an optional out-of-core runner under `runners/openai`. It speaks the fbt
stdio JSON-RPC protocol, reads `OPENAI_API_KEY` from the environment, calls the
OpenAI Responses API, writes output candidates under `work.outputs`, and keeps
provider logic outside `internal/` and the base CLI.

## Permanent Fix

Implemented `runners/openai` with unit coverage against an `httptest` Responses
API fixture. Added project-local `bin/fbt-runner-openai` wrappers to the
practical examples so source-checkout users can run real provider builds after
exporting `OPENAI_API_KEY`.

## Next Check

Run:

```sh
make verify
```
