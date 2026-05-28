# dbt Core Overview Report

Created: 2026-05-28  
Audience: fbt design research

## 1. Executive Summary

dbt's core advantage is not simply running SQL. It turned the SQL transformation layer into a software-engineering workflow: declarative projects, dependency management, DAG execution, tests, documentation, artifacts, packages, and adapter plugins.

For `fbt`, the key lesson is that the product should not be a one-off runner. The value is a coherent control plane around resources, dependency graphs, state, quality checks, documentation, and an ecosystem boundary.

## 2. What Is a dbt Project?

A dbt project is a repository containing SQL models, sources, tests, macros, seeds, snapshots, docs, and configuration.

Typical layout:

```text
dbt_project.yml
models/
macros/
seeds/
snapshots/
tests/
analyses/
packages.yml
target/
```

Core ideas:

- `ref()` expresses dependencies between models.
- `source()` references external warehouse sources.
- Models are selected and run as a DAG.
- Tests and docs are part of the transformation workflow.
- Artifacts such as `manifest.json` and `run_results.json` make the project inspectable and extensible.

## 3. dbt-core Repository Structure

Important directories in `dbt-core`:

| Path | Purpose |
|---|---|
| `core/dbt/cli` | Click-based CLI entry point |
| `core/dbt/config` | Project, profile, and runtime configuration |
| `core/dbt/parser` | Manifest, model, source, macro, and YAML parsing |
| `core/dbt/contracts` | Graph contracts and runtime node types |
| `core/dbt/artifacts` | Artifact schema and resource definitions |
| `core/dbt/graph` | DAG selection and graph methods |
| `core/dbt/task` | Command implementations |
| `core/dbt/materializations` | Materialization strategies |
| `core/dbt/events` | Structured event logging |
| `tests/` | Unit and functional tests |
| `plugins/` | Local adapter plugins for tests |
| `schemas/` | JSON schemas for artifacts |

## 4. Processing Flow

### CLI

The `dbt` CLI dispatches commands such as `run`, `build`, `test`, `compile`, and `docs`.

### Parsing

Parsing reads project files and produces a manifest. This is separate from execution and enables docs, state comparison, and partial parsing.

### Execution

Execution selects nodes, orders them by dependencies, and delegates database-specific work to adapters and materializations.

### Artifacts

Artifacts such as `manifest.json` and `run_results.json` are central extension points. External tools can inspect lineage, timing, tests, and state.

## 5. Sources of dbt's Advantage

### SQL-first adoption

dbt uses SQL, a familiar language for analysts and analytics engineers.

### `ref()` and DAG semantics

Logical references decouple physical object names from dependency definitions.

### Integrated tests

Data quality checks are part of the transformation workflow rather than an afterthought.

### Materialization abstraction

Users define logical models; adapters and materializations control how they become tables, views, incremental models, or snapshots.

### Adapter ecosystem

Adapter boundaries let dbt support many warehouses without putting warehouse-specific logic in core.

### Package ecosystem

Reusable macros, tests, and models made dbt more than a local CLI.

### Artifacts

Machine-readable artifacts enabled documentation, state selection, CI, lineage, and downstream tooling.

### State, defer, and partial parsing

State-based workflows made dbt efficient and usable in CI and production.

## 6. Adjacent Tools

| Tool | Comparison |
|---|---|
| Dataform | Strong in BigQuery/GCP contexts; narrower ecosystem |
| SQLMesh | Strong virtual environments, planning, SQL understanding, and table diff |
| Airflow / Dagster / Prefect | Workflow orchestrators; broader but heavier and task-centric |
| Matillion / low-code ELT | Useful for visual workflows; less code-native and artifact-centric |

## 7. dbt Platform Value

Beyond dbt-core, the platform adds:

- hosted scheduling
- environments
- metadata and docs
- CI
- governance
- collaboration
- observability

The important lesson for `fbt` is to keep the local core lightweight while leaving a natural path to managed collaboration and governance.

## 8. Where dbt Is Strong

dbt is a strong fit when:

- transformations are SQL-centric
- teams want code review and version control
- lineage and docs matter
- tests and CI are needed
- adapter and package ecosystem matter

## 9. Where dbt Is Weaker

dbt is less ideal when:

- transformations are not SQL-shaped
- workflows require arbitrary orchestration
- strict virtual environments or semantic SQL validation are the top priority
- unstructured documents, LLM outputs, and agentic transformations dominate

## 10. Implications for fbt

`fbt` should borrow:

- project convention
- resource graph
- `ref()` and `source()`
- artifacts
- state comparison
- docs generation
- plugin boundary
- test/eval integration

`fbt` should not copy:

- warehouse relation assumptions
- SQL as the transformation language
- adapter/materialization semantics too literally
- a heavy platform requirement in the base tool

## 11. References

- dbt documentation: https://docs.getdbt.com/
- dbt packages: https://docs.getdbt.com/docs/build/packages
- dbt artifacts: https://docs.getdbt.com/reference/artifacts/dbt-artifacts

