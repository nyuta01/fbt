# Internal Package Boundaries

`internal/` contains Go implementation packages for `fbt-core`. Product
packages should be added only when backed by an accepted spec or execution-plan
task. The current codebase includes the CLI scaffold, project/config/resource
parser baseline, manifest resource generation, selector helpers, artifact
descriptor computation, reusable path-safety helpers, and a local filesystem
state store. The templates package scaffolds local project examples. The
planner baseline can compare manifests and state snapshots to classify
transforms as run, skip, or blocked. The public CLI exposes the small project
loop: init, doctor, plan, build, artifact inspection, diff, and standard
exports. The runner and protocol packages discover external runner commands,
start JSON-RPC stdio runners, and collect events/output candidates.
Runner adapter examples live outside `internal/` under
`examples/runner_adapters/`; protocol-only test fixtures live under
`tests/runner_fixtures/`. The build package wires the current
parse-plan-run-commit-state lifecycle for local protocol runners, with baseline
policy checks, deterministic evals, and runner usage/provenance records before
official commit.

Package boundaries:

| Package | Responsibility |
|---|---|
| `project` | Project discovery, `fs_project.yml`, path defaults |
| `config` | YAML decoding, validation, defaults, config versioning |
| `parser` | Resource-file parsing and diagnostics |
| `manifest` | Parsed graph resources and manifest serialization |
| `graph` | Dependency graph, selectors, parent and child maps |
| `planner` | Dirty-state comparison and build plan generation |
| `build` | Runner execution, evals, policy checks, and official commits |
| `state` | Local state store, locks, run results, artifact versions |
| `templates` | Local project scaffolds for `fbt init` |
| `artifact` | Descriptor computation, artifact versions, commit boundary |
| `runner` | Runner discovery, process lifecycle, protocol client |
| `eval` | Deterministic and delegated eval orchestration |
| `policy` | Policy decisions and write-scope checks |
| `security` | Path containment and symlink safety helpers |
| `diff` | Raw text and Markdown heading-aware artifact diffs |
| `lineage` | OpenLineage export construction |
| `telemetry` | OTLP/JSON export construction |
| `plugin` | Runner/plugin manifest handling, not in-process execution |
| `protocol` | JSON-RPC message types and compatibility checks |
| `cli` | Cobra command wiring and user-facing output |
| `version` | Build and release version metadata |

Do not place LLM provider clients, document converters, OCR engines, or agent
runtimes in core packages. Those belong to external runners.
