# fbt Runner Protocol Spec

Status: Draft  
Created: 2026-05-28  
Audience: implementers of `fbt-core` and external runners

## 1. Overview

`fbt` core does not implement transform logic. It parses projects, plans work, invokes runners, evaluates outputs, manages review gates, and records state. Runners execute transforms.

The initial runner protocol is **JSON-RPC 2.0 compatible messages over stdio**. Runners may be implemented in Go, Python, TypeScript, Rust, shell, or internal binaries.

## 2. Design Principles

### Necessary, Not Minimal

The protocol must be sufficient for LLM and agent transforms, not just command execution. It includes:

- Capability negotiation
- Transform request / response
- Artifact input resolution
- Declared output contract
- Policy and writable scope
- Transform assets, model, and tool metadata
- Progress and trace events
- Token usage and cost
- Tool-call log
- Output candidate declaration
- Warnings and errors
- Commit boundary for idempotent core commit

### Core Owns State, Runner Owns Execution

Core owns:

- Manifest
- Graph
- State
- Artifact descriptors and digests
- Official commit
- Approval state
- Docs and lineage
- Runner invocation lifecycle

Runner owns:

- Transform execution
- Model, agent, command, or converter invocation
- Intermediate tool calls
- Output generation into assigned work directories
- Runner-specific trace details

Runners never commit official artifact state.

### JSON-RPC Compatible, Not LSP

The protocol follows JSON-RPC request/response/notification/error semantics and borrows LSP-style patterns:

- `initialize`
- cancellation
- progress events
- request IDs
- structured errors

### No In-Process Plugins

MVP runners are separate processes to avoid dependency and runtime conflicts in core.

Discovery and future plugin installation semantics are defined in
[Runner Discovery Spec](runner-discovery-spec.md).

### Stable Envelope, Extensible Payload

The JSON-RPC envelope is stable. Provider-specific details live in `metadata`, `attributes`, or `extensions`.

## 3. Transport and Framing

Initial transport: stdio.

- Core starts runner process.
- Core writes JSON-RPC messages to stdin.
- Runner writes JSON-RPC messages to stdout.
- stderr is human-readable debug output.
- MVP framing is JSON Lines: one JSON-RPC object per line.
- Future versions may add LSP-style `Content-Length` framing.

```text
fbt-core  --stdin JSON-RPC JSONL-->  runner
fbt-core  <--stdout JSON-RPC JSONL-- runner
```

## 4. Message Model

| Category | Direction | Purpose |
|---|---|---|
| request | core -> runner | `initialize`, `fbt/runTransform`, `fbt/validate` |
| response | runner -> core | request result or error |
| notification | core -> runner | `initialized`, `$/cancelRequest` |
| notification | runner -> core | `fbt/event`, `fbt/outputCandidate`, `fbt/heartbeat` |

Methods:

| Method | Type | Purpose |
|---|---|---|
| `initialize` | request | Protocol and capability negotiation |
| `initialized` | notification | Initialization complete |
| `fbt/runTransform` | request | Execute transform |
| `fbt/validate` | request | Validate request or dry-run |
| `fbt/event` | notification | Progress, trace, tool call, usage |
| `fbt/outputCandidate` | notification | Declare generated output candidates |
| `fbt/heartbeat` | notification | Liveness |
| `$/cancelRequest` | notification | Cancellation |

## 5. Common JSON-RPC Shape

Request:

```json
{
  "jsonrpc": "2.0",
  "id": "req_01H...",
  "method": "fbt/runTransform",
  "params": {}
}
```

Response:

```json
{
  "jsonrpc": "2.0",
  "id": "req_01H...",
  "result": {}
}
```

Notification:

```json
{
  "jsonrpc": "2.0",
  "method": "fbt/event",
  "params": {}
}
```

Error:

```json
{
  "jsonrpc": "2.0",
  "id": "req_01H...",
  "error": {
    "code": -32010,
    "message": "Policy denied",
    "data": {
      "fbt_error_code": "POLICY_DENIED",
      "retryable": false,
      "details_redacted": {}
    }
  }
}
```

## 6. Initialize

Core sends:

```json
{
  "jsonrpc": "2.0",
  "id": "init_001",
  "method": "initialize",
  "params": {
    "core": {
      "name": "fbt-core",
      "version": "0.1.0"
    },
    "protocol": {
      "versions": ["0.1"],
      "framing": "jsonl",
      "schema_version": "https://schemas.fbt.dev/runner-protocol/v0.1.json"
    },
    "capability_request": [
      "run_transform",
      "stream_events",
      "tool_call_log",
      "usage_reporting",
      "output_candidates",
      "cancellation"
    ]
  }
}
```

Runner responds:

```json
{
  "jsonrpc": "2.0",
  "id": "init_001",
  "result": {
    "runner": {
      "id": "runner.knowledge_ops.openai.responses",
      "name": "fbt-openai",
      "version": "0.1.0",
      "language": "typescript"
    },
    "protocol": {
      "version": "0.1",
      "framing": "jsonl"
    },
    "capabilities": {
      "transform_types": ["llm"],
      "artifact_types": ["markdown", "markdown_directory", "text"],
      "stream_events": true,
      "tool_call_log": false,
      "usage_reporting": true,
      "cost_estimation": true,
      "supports_dry_run": true,
      "supports_cancel": true
    }
  }
}
```

## 7. Run Transform

`fbt/runTransform` executes a transform. Progress, usage, tool calls, and output candidates may stream as notifications.

Request includes:

- `mode`: `plan`, `dry_run`, `run`, or `eval`
- invocation and transform run IDs
- trace context
- transform identity and config
- resolved inputs
- declared outputs
- transform assets
- model parameters
- tools
- policy
- previous state references
- scoped work directories

Skeleton:

```json
{
  "jsonrpc": "2.0",
  "id": "req_123",
  "method": "fbt/runTransform",
  "params": {
    "mode": "run",
    "invocation_id": "inv_01H...",
    "transform_run_id": "transform_run.run_01H...",
    "transform": {
      "unique_id": "transform.knowledge_ops.contract_summaries",
      "name": "contract_summaries",
      "type": "llm",
      "fingerprint": "sha256:transform..."
    },
    "inputs": [],
    "outputs": [],
    "assets": [],
    "model": {},
    "tools": [],
    "policy": {},
    "state": {},
    "work": {
      "root": "/repo/.fbt/work/req_123",
      "temp": "/repo/.fbt/work/req_123/tmp",
      "outputs": "/repo/.fbt/work/req_123/outputs"
    }
  }
}
```

## 8. Event Notification

Runner event:

```json
{
  "jsonrpc": "2.0",
  "method": "fbt/event",
  "params": {
    "request_id": "req_123",
    "transform_run_id": "transform_run.run_01H...",
    "time": "2026-05-28T10:00:10Z",
    "event_type": "usage",
    "level": "info",
    "message": "LLM request completed",
    "attributes": {
      "gen_ai.provider.name": "openai",
      "gen_ai.request.model": "gpt-5",
      "gen_ai.usage.input_tokens": 12000,
      "gen_ai.usage.output_tokens": 1800,
      "fbt.estimated_cost_usd": 0.42
    }
  }
}
```

Event types:

- `progress`
- `log`
- `usage`
- `tool_call.started`
- `tool_call.completed`
- `artifact.observed`
- `retrieval.completed`
- `warning`
- `debug`

Raw prompts, inputs, and outputs should not be emitted by default.

## 9. Tool Call Event

```json
{
  "jsonrpc": "2.0",
  "method": "fbt/event",
  "params": {
    "request_id": "req_agent_001",
    "transform_run_id": "transform_run.run_agent_001",
    "event_type": "tool_call.completed",
    "attributes": {
      "gen_ai.tool.call.id": "tool_001",
      "gen_ai.tool.name": "read_artifact",
      "gen_ai.tool.type": "function",
      "fbt.tool.status": "success"
    },
    "tool_call": {
      "id": "tool_001",
      "name": "read_artifact",
      "arguments_redacted": {
        "artifact": "contract_summaries"
      },
      "status": "success"
    }
  }
}
```

Tool-call logs are for audit and must redact credentials.

## 10. Output Candidate Notification

Runners write outputs to assigned work directories and declare candidates.

```json
{
  "jsonrpc": "2.0",
  "method": "fbt/outputCandidate",
  "params": {
    "request_id": "req_123",
    "transform_run_id": "transform_run.run_01H...",
    "outputs": [
      {
        "name": "contract_summaries",
        "unique_id": "artifact.knowledge_ops.contract_summaries",
        "artifact_type": "markdown_directory",
        "path": "/repo/.fbt/work/req_123/outputs/contract_summaries/",
        "declared_path": "target/artifacts/contracts/summaries/",
        "metadata": {
          "file_count": 12
        }
      }
    ]
  }
}
```

Authoritative descriptors and digests are computed by core.

## 11. Run Transform Response

```json
{
  "jsonrpc": "2.0",
  "id": "req_123",
  "result": {
    "status": "success",
    "transform_run_id": "transform_run.run_01H...",
    "outputs": [
      {
        "name": "contract_summaries",
        "unique_id": "artifact.knowledge_ops.contract_summaries",
        "path": "/repo/.fbt/work/req_123/outputs/contract_summaries/",
        "artifact_type": "markdown_directory"
      }
    ],
    "usage": {
      "gen_ai.usage.input_tokens": 12000,
      "gen_ai.usage.output_tokens": 1800,
      "fbt.usage.total_tokens": 13800,
      "fbt.estimated_cost_usd": 0.42
    },
    "provenance": {
      "runner": "runner.knowledge_ops.openai.responses",
      "runner_version": "0.1.0",
      "model_provider": "openai",
      "model": "gpt-5",
      "model_parameters_hash": "sha256:params...",
      "materials": []
    },
    "warnings": []
  }
}
```

Status values:

- `success`
- `failed`
- `cancelled`
- `skipped`
- `blocked`

## 12. Error Handling

Use standard JSON-RPC error codes where possible. fbt-specific errors use `-32099` to `-32000` and `error.data.fbt_error_code`.

Initial fbt error codes:

- `INVALID_REQUEST`
- `MISSING_INPUT`
- `POLICY_DENIED`
- `TIMEOUT`
- `COST_LIMIT_EXCEEDED`
- `MODEL_ERROR`
- `MODEL_RATE_LIMITED`
- `TOOL_ERROR`
- `OUTPUT_CONTRACT_FAILED`
- `INTERNAL_ERROR`

## 13. Cancellation

```json
{
  "jsonrpc": "2.0",
  "method": "$/cancelRequest",
  "params": {
    "id": "req_123",
    "reason": "user_cancelled"
  }
}
```

Runners should stop work, clean runner-owned temporary resources, and respond with either `status: cancelled` or a structured cancellation error.

## 14. Security Requirements

Core requirements:

- Pass scoped paths to runners
- Require outputs under work directories
- Pass only necessary secrets
- Never store secrets in manifest or run results
- Validate structured stdout
- Treat stderr as untrusted human logs
- Enforce timeouts, cost limits, and output-size limits
- Reject output candidates outside the invocation work directory

Runner requirements:

- Respect policy
- Do not print secrets to structured stdout
- Do not write outside declared write paths
- Redact credentials in tool-call logs
- Return fatal errors as structured errors

The full MVP security boundary and fake-runner conformance suite are defined in
[Security and Conformance Spec](security-and-conformance-spec.md).

## 15. Commit Boundary

```text
runner output candidate
  -> fbt-core descriptor / digest
  -> fbt-core eval orchestration
  -> fbt-core policy / approval check
  -> fbt-core immutable artifact_version record
  -> fbt-core logical pointer update
```

This keeps official state safe across retries, failures, and interruptions.

## 16. Dry Run and Cost Estimate

LLM and agent transforms need a good planning experience.

In `plan` or `dry_run` mode, runners may return:

- estimated tokens
- estimated cost
- expected output count
- missing capabilities
- policy risks
- whether the runner supports the requested artifact types

## 17. Versioning

Protocol versioning:

- `0.x`: draft protocol
- `1.x`: stable envelope

Runners return supported protocol versions in `initialize`. Core rejects incompatible runners clearly.

Static runner manifests are advisory. The `initialize` response is authoritative
for the current process. Authoritative artifact descriptors and digests are
computed by core, not by runners.

## 18. MVP Required Capabilities

Required:

- `initialize`
- `fbt/runTransform`
- output candidate declaration
- success/failure response
- structured error response
- cancellation
- basic usage reporting for AI runners
- scoped output directories

Not required in MVP:

- remote worker transport
- binary artifact streaming over protocol
- gRPC
- plugin marketplace

## 19. Remaining Protocol Decisions

MVP is fixed as JSON-RPC 2.0 compatible messages over stdio, JSONL framing,
runner process isolation, project/plugin/PATH discovery, and core-owned
descriptors. Remaining decisions:

1. Whether `v1` keeps JSONL framing or adds LSP-style `Content-Length` framing.
2. How precise cost estimates must be as protocol contract.
3. How far agent tool-call logs should align with OpenTelemetry spans and GenAI semantic conventions.
4. When to introduce remote runner transport and how it stays compatible with stdio.
5. Whether approval, eval, and storage providers use the same JSON-RPC protocol family or separate provider protocols.
