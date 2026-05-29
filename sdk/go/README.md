# fbt Go Runner SDK

`sdk/go` is the provider-free helper module for external fbt runner adapters.

It contains:

- protocol types for fbt runner JSON-RPC messages
- a small JSONL stdio JSON-RPC server helper
- output-candidate helpers
- redaction helpers

The SDK is not the source of truth. The runner protocol spec and conformance
suite remain authoritative:

```sh
python3 tests/runner-conformance/run.py --runner-command 'adapter-command' --strict
```

The SDK must not depend on provider SDKs, agent CLIs, fbt core `internal/`
packages, or adapter implementations.
