# Internal Package Boundaries

`internal/` contains Go implementation packages for `fbt-core`. Product
packages should be added only when backed by an accepted spec or execution-plan
task. The current codebase includes the CLI scaffold, project/config/resource
parser baseline, manifest resource generation, selector helpers, artifact
descriptor computation, reusable path-safety helpers, and a local filesystem
state store. The templates package scaffolds local project examples. The
planner baseline can compare manifests and state snapshots to classify
transforms as run, skip, or blocked. The CLI now exposes init, parse, plan,
build, eval, review, state, artifact, and runner diagnostics. The protocol
package can start JSON-RPC stdio runners and collect events/output candidates.
Local fake and command runners live outside `internal/` under `runners/`,
alongside deterministic demo LLM and agent runner examples. The build package wires
the current
parse-plan-run-commit-state lifecycle for local protocol runners, with baseline
policy checks, deterministic evals, pending-review approval state, and runner
usage/provenance records before official commit.

Package boundaries:

| Package | Responsibility |
|---|---|
| `project` | Project discovery, `fs_project.yml`, path defaults |
| `config` | YAML decoding, validation, defaults, config versioning |
| `manifest` | Parsed graph resources and manifest serialization |
| `graph` | Dependency graph, selectors, parent and child maps |
| `planner` | Dirty-state comparison and build plan generation |
| `state` | Local state store, locks, run results, approvals |
| `templates` | Local project scaffolds for `fbt init` |
| `artifact` | Descriptor computation, artifact versions, commit boundary |
| `runner` | Runner discovery, process lifecycle, protocol client |
| `eval` | Deterministic and delegated eval orchestration |
| `approval` | Review gates and artifact-version approval state |
| `diff` | Raw text and Markdown heading-aware artifact diffs |
| `docs` | Static lineage and review documentation generation |
| `plugin` | Runner/plugin manifest handling, not in-process execution |
| `protocol` | JSON-RPC message types and compatibility checks |

Do not place LLM provider clients, document converters, OCR engines, or agent
runtimes in core packages. Those belong to external runners.
