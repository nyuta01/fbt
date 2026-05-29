# FBT-RUNNER-020 Fix OpenAI live conformance model handling

## Observation

OpenAI live conformance uses the shared runner conformance request, whose model
metadata is `provider: conformance` and `name: fixture`. The OpenAI adapter
previously forwarded that fixture model name to the real Responses API, which
would make live provider checks fail even when credentials are valid.

## Decision

Treat conformance fixture model metadata as test metadata, not as a real
provider model. The adapter now falls back to `FBT_OPENAI_DEFAULT_MODEL` or
`gpt-5` when the request model is the conformance fixture.

## Permanent Fix

OpenAI adapter unit tests and network-free conformance still pass. Live OpenAI
adapter conformance passed with a real `OPENAI_API_KEY`. A temporary copy of
`examples/incident_response_runbook` also completed
`doctor -> plan -> build -> artifact show` with the real OpenAI adapter.

## Next Check

Done. `make verify` passes. The API key was not written to repository files or
printed in command output.
