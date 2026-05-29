# fbt-runner-openai

Official OpenAI Responses adapter for fbt.

`fbt-runner-openai` turns fbt `type: llm` transform requests into OpenAI
Responses API calls, writes generated content under `work.outputs`, and reports
output candidates back to fbt.

The adapter reads credentials from `OPENAI_API_KEY`. fbt core stores only the
environment variable name in project configuration; it does not store or print
the secret value.

## Development

From the repository root:

```sh
cd adapters/openai
go test ./...
```

Protocol conformance without a live provider call:

```sh
OPENAI_API_KEY=test \
FBT_OPENAI_ADAPTER_FAKE_RESPONSE='# OpenAI Adapter Conformance' \
python3 tests/runner-conformance/run.py \
  --runner-command 'go run ./adapters/openai/cmd/fbt-runner-openai' \
  --transform-type llm \
  --strict
```

`FBT_OPENAI_ADAPTER_FAKE_RESPONSE` is a test-only bypass for conformance and
CI. Normal fbt projects should omit it so the adapter calls the Responses API.
