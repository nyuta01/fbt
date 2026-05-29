# fbt Runner Lockfile Spec

Status: MVP-ready
Created: 2026-05-29
Audience: teams that need reproducible runner and adapter environments

## 1. Purpose

`fbt.lock.json` is an optional project-local reproducibility contract for
external runners and official adapter packages. It records what the project
expects to execute; it does not install, update, or resolve packages.

The lockfile exists for teams that want CI, agents, and developers to notice
runner drift before generated artifacts change unexpectedly.

## 2. Non-Goals

Core must not become a package manager. Reading a lockfile must not:

- download runners or adapters
- mutate `fs_project.yml`
- rewrite plugin manifests
- choose newer compatible versions
- contact registries or provider APIs
- load adapter code into the core process

Installation remains out-of-band through `go install`, package managers,
checked-in plugin directories, container images, CI setup, or organization
tooling.

## 3. File Location

When used, the lockfile is named:

```text
fbt.lock.json
```

It lives at the project root next to `fs_project.yml`. It should be committed
when a team wants runner reproducibility across machines.

## 4. Schema

Schema URI:

```text
https://schemas.fbt.dev/fbt/runner-lock/v1.json
```

Minimal shape:

```json
{
  "fbt_schema_version": "https://schemas.fbt.dev/fbt/runner-lock/v1.json",
  "lockfile_version": 1,
  "runners": {
    "openai.responses": {
      "source": "github.com/nyuta01/fbt/adapters/openai",
      "version": "adapters/openai/v0.1.0",
      "protocol_version": "0.1",
      "command": "fbt-runner-openai",
      "manifest_digest": "sha256:...",
      "checksums": {
        "go_module": "h1:...",
        "darwin-arm64": "sha256:..."
      },
      "capabilities": {
        "transform_types": ["llm"],
        "artifact_types": ["markdown", "markdown_directory"],
        "output_candidates": true
      }
    }
  }
}
```

Fields:

| Field | Required | Meaning |
|---|---:|---|
| `fbt_schema_version` | yes | Lockfile schema URI. |
| `lockfile_version` | yes | Integer lockfile semantics version. MVP is `1`. |
| `runners` | yes | Map keyed by logical runner name from project config or plugin manifest. |
| `source` | no | Human-readable adapter source address, module path, package name, or internal source ID. |
| `version` | no | Adapter release tag, module version, package version, or internal release ID. |
| `protocol_version` | yes | Expected fbt runner protocol version. |
| `command` | no | Expected executable name or project-relative command. |
| `manifest_digest` | no | Digest of the plugin manifest or adapter manifest used to resolve the runner. |
| `checksums` | no | Checksums by channel, platform, module checksum, or binary artifact name. |
| `capabilities` | no | Expected capabilities used as a doctor/conformance expectation. |

Lock entries may contain `meta` for organization-owned annotations. Core must
preserve but not interpret `meta`.

## 5. Core Behavior When Present

Core behavior is validation and explanation only:

- `fbt doctor` parses `fbt.lock.json` and reports malformed JSON,
  unsupported schema versions, unknown logical runners, unused lock entries, and
  mismatches between locked expectations and resolved runner identity.
- `fbt doctor` reports checksum expectations when it can compare them locally,
  such as command or plugin manifest checksums, but it must not download missing
  checksums or contact registries.
- `fbt plan` and `fbt build` treat the lockfile entry digest as part of runner
  identity. A matching lock entry change makes dependent transforms dirty.
- `fbt build` fails before `fbt/runTransform` when a present lockfile says the
  selected runner is incompatible with the resolved command, local checksum,
  negotiated protocol version, or negotiated capabilities.
- Manifest runner resources expose lockfile-derived runner identity when a
  valid matching lock entry is available.

The absence of `fbt.lock.json` is valid. Projects can stay lightweight and rely
on runner config, plugin manifests, `PATH`, and conformance checks.

## 6. Doctor Diagnostics

Diagnostic codes:

| Code | Severity | Meaning |
|---|---|---|
| `RUNNER_LOCK_OK` | info | Present lock entry matches resolved runner identity. |
| `RUNNER_LOCK_UNUSED` | warning | Lock entry has no matching configured runner. |
| `RUNNER_LOCK_MISSING` | warning | Configured runner has no lock entry when at least one lock entry exists. |
| `RUNNER_LOCK_SCHEMA_UNSUPPORTED` | error | Lockfile schema or version is unsupported. |
| `RUNNER_LOCK_MISMATCH` | error | Command, protocol, manifest digest, checksum, or capability expectation does not match. |

These diagnostics are part of `doctor`; they are not install or update actions.

## 7. Conformance Expectations

Conformance fixtures prove:

- a valid lockfile does not require network access
- malformed or unsupported lockfiles fail deterministically
- unused entries and missing entries are visible in `doctor`
- command/protocol/capability mismatches fail before runner execution
- lockfile changes participate in dirty-state selection
- no core command downloads, installs, or resolves adapter packages

## 8. Relationship To Other Files

| File | Role |
|---|---|
| `fs_project.yml` | Declares logical runner names and direct runner config. |
| `fbt_plugin.yml` | Declares adapter-provided runner commands and capabilities. |
| `fbt.lock.json` | Pins expected runner/adapter identity and local integrity metadata. |
| `.fbt/state/manifest.json` | Records the parsed graph and current resolved fingerprints. |
| `.fbt/state/run_results.jsonl` | Records what actually ran. |

The lockfile is an optional guardrail around external runner identity. It never
replaces the runner protocol handshake or conformance suite.
