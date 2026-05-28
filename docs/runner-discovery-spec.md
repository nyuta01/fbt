# fbt Runner Discovery Spec

Status: Draft  
Created: 2026-05-28  
Audience: implementers of runner configuration, discovery, diagnostics, and future plugin installation

## 1. Overview

`fbt` core delegates transformation execution to external runners. Runner
discovery must keep the base tool local-first and lightweight while making LLM
and agent runners easy to use.

MVP does not include an in-process plugin system or network package manager.
Runners are ordinary executables discovered from project configuration, plugin
manifests, or `PATH`.

## 2. Runner Identity

Transforms reference runners by logical name:

```yaml
runner: openai.responses
```

The manifest expands this into:

```text
runner.<project>.openai.responses
```

Runner identity used for dirty-state comparison includes:

- logical runner name
- resolved executable path or plugin manifest identity
- executable fingerprint when available
- protocol version
- negotiated capabilities
- runner configuration hash
- model and tool configuration relevant to the transform

## 3. Resolution Order

Core resolves a runner reference in this order:

1. `runners` entry in `fs_project.yml` with an explicit `command`.
2. Project-local plugin manifest under `plugins/*/fbt_plugin.yml`.
3. User-local plugin manifest under `${FBT_HOME:-~/.fbt}/plugins/*/fbt_plugin.yml`.
4. `PATH` lookup using the conventional executable name
   `fbt-runner-<normalized-runner-name>`, where dots and underscores become
   hyphens.

The first matching precedence level wins. Multiple matches at the same
precedence level are an error.

Relative commands in project config are resolved from the project directory.
Absolute commands are allowed but reduce portability and should be reported by
`fbt runner doctor`.

## 4. Project Runner Config

Canonical project config:

```yaml
runners:
  - name: openai.responses
    type: llm
    protocol: stdio_jsonrpc
    command: fbt-openai-runner
    args: ["--profile", "fbt"]
    cwd: .
    env:
      - OPENAI_API_KEY
    config:
      provider: openai
      default_model: gpt-5
```

Fields:

| Field | Required | Meaning |
|---|---:|---|
| `name` | yes | Logical runner name used by transforms |
| `type` | yes | `command`, `extract`, `template`, `llm`, `agent`, `eval`, or `converter` |
| `protocol` | yes | `stdio_jsonrpc` for MVP |
| `command` | no | Executable name or path; required unless a plugin manifest provides it |
| `args` | no | Static process arguments passed after `command` |
| `cwd` | no | Runner working directory; project config paths resolve from the project directory |
| `env` | no | Environment variable names passed through by core; values are never stored or printed |
| `config` | no | Runner-specific configuration included in fingerprints |
| `capabilities` | no | Static expected capabilities; verified by `initialize` |

If `cwd` is omitted, core leaves the runner process working directory unchanged
from the parent `fbt` process. This preserves existing wrappers while allowing
projects to opt into explicit working directories. Core passes a small base
environment (`PATH`, `HOME`, user and temp directory variables when present)
plus the declared `env` names. Missing declared environment variables are
reported by `fbt runner doctor` and `fbt doctor` without printing values.

## 5. Plugin Manifest

Plugin manifests are metadata, not in-process code.

`plugins/openai/fbt_plugin.yml`:

```yaml
name: fbt-openai
version: 0.1.0
protocol: stdio_jsonrpc
command: fbt-openai-runner
args: ["--profile", "fbt"]
cwd: .
provides:
  - runner: openai.responses
    type: llm
    transform_types: ["llm"]
    artifact_types: ["markdown", "markdown_directory", "text"]
env:
  - OPENAI_API_KEY
checksum:
  command: sha256:...
```

Core reads plugin manifests to resolve commands and expected capabilities. It
does not load plugin code into the core process.

For plugin manifests, relative `command` and `cwd` values resolve from the
plugin manifest directory. If plugin `cwd` is omitted, core leaves the parent
working directory unchanged.

## 6. Capability Negotiation

Static config and plugin manifests are advisory. The `initialize` response from
the running process is authoritative for the current invocation.

Core must reject a runner when:

- protocol version is incompatible
- required transform type is unsupported
- required artifact type is unsupported
- required events or output-candidate capability is missing
- runner identity conflicts with the selected runner reference

Runner incompatibility exits with code `6` when detected before invocation and
code `4` when detected as a protocol error during invocation.

## 7. Plugin Installation Semantics

MVP does not download or install plugins. Installation is out-of-band through
the host environment, such as `go install`, `brew`, `npm`, `uv tool`, or a
checked-in `plugins/` directory.

Reserved future command:

```sh
fbt plugin install SOURCE [--version VERSION] [--project | --user] [--save]
```

When implemented, `plugin install` must:

- verify a declared digest or signature when provided
- install into a project-local or user-local plugin directory
- write or update a plugin manifest
- avoid mutating `fs_project.yml` unless `--save` is passed
- never introduce in-process code loading into core
- make the installed runner visible to `fbt runner list`

Until then, users validate availability with:

```sh
fbt runner list
fbt runner doctor openai.responses
fbt runner validate openai.responses
```

## 8. Diagnostics

`fbt runner list` shows:

- logical runner name
- resolution source
- command
- args, cwd, and env names when configured
- negotiated status if recently validated

`fbt runner doctor RUNNER` checks:

- command exists and is executable
- configured cwd exists and is a directory
- declared environment variable names are present without printing values
- plugin manifest shape is valid
- `initialize` succeeds
- capabilities satisfy configured transforms

`fbt runner validate RUNNER` performs protocol validation and exits non-zero on
incompatibility.

## 9. Locking and Reproducibility

Core records runner identity in manifest, run results, and transform effective
fingerprints. A runner identity change marks dependent transforms dirty.

MVP does not require a committed lockfile. A future `fbt.lock.json` may pin
runner package source, version, and digest for teams that need stronger
reproducibility.
