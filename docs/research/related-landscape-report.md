# fbt Related Landscape Report

Created: 2026-05-28  
Audience: fbt positioning and product strategy

## 1. Conclusion

`fbt` sits at the intersection of file/DAG pipelines, document processing, LLM/agent workflows, eval/observability, and lineage/provenance systems. The closest adjacent concept is DataChain, but `fbt` is differentiated by a dbt-like build control plane for filesystem artifacts, LLM/agent transform management, artifact versions, evals, approval, and local-first CLI ergonomics.

## 2. Landscape Map

| Category | Examples | Relevance |
|---|---|---|
| File/DAG pipeline | DVC, Snakemake, Nextflow, Dagster | Dependency graph and file outputs |
| Data/file versioning | Pachyderm, lakeFS, Git LFS, git-annex | Versioned artifacts and provenance |
| Document extraction | Unstructured, Apache Tika, Pandoc, Quarto | Runners/plugins, not core |
| Markdown/content transform | unified, remark, rehype, retext | Strong reference for Markdown transforms |
| RAG/document pipeline | LlamaIndex, Haystack | Document processing and retrieval workflows |
| Agent workflows | LangGraph, CrewAI, AutoGen, OpenAI Agents SDK, DSPy | Agent execution boundary |
| Eval/observability | Langfuse, LangSmith, Braintrust, Promptfoo, MLflow GenAI, Weave | Evals, trace, prompt/model metadata |
| Lineage/protocol | OpenLineage, OpenTelemetry, MCP | Export and integration standards |
| Security/governance | Presidio, Guardrails | Validation and policy references |

## 3. Closest Reference: DataChain

DataChain is close because it targets unstructured data and AI workflows. It emphasizes versioned datasets, schemas, lineage, and agent-readable data context.

DataChain strengths:

- Strong unstructured-data positioning
- Versioned datasets
- Schema-aware data handling
- Agent context story
- Cloud/object storage orientation

Potential fbt differentiation:

- dbt-like project and build workflow
- filesystem artifact graph, not only datasets
- transform assets, policies, evals, and approval as graph concepts
- local-first single-command UX
- first-class LLM/Agent transformation contract
- artifact_version / transform_run separation
- human review and downstream blocking semantics

## 4. File/DAG Pipeline Tools

### DVC

DVC defines stages with commands, dependencies, and outputs. It records hashes in lock files and is strong for ML/data pipelines.

`fbt` should borrow explicit dependency tracking and dirty-state semantics, but focus on document artifacts, LLM/agent transforms, evals, approval, and lineage.

### Snakemake / Nextflow

These are powerful workflow systems for reproducible pipelines. They are strong in scientific and data-processing workflows.

`fbt` should avoid becoming a general workflow orchestrator. Its center should remain artifacts and transformation contracts.

### Dagster

Dagster is asset-oriented and offers strong orchestration, observability, and data platform capabilities.

`fbt` should be lighter, local-first, and file-artifact-specific.

## 5. Data and File Versioning

### Pachyderm / lakeFS

These tools provide strong versioning and provenance over data repositories or object storage.

`fbt` can learn from immutable versioning and provenance, but should not require a storage platform in the base tool.

### Git LFS / git-annex

Useful references for large-file handling, but they do not manage transform contracts, evals, or approvals.

## 6. Document Extraction and Conversion

### Unstructured

Strong for partitioning and extracting content from many document formats. It should be a runner/plugin candidate, not core.

### Apache Tika

Broad metadata and text extraction support. Useful as an extraction runner.

### Pandoc / Quarto

Excellent references for document conversion and publishing. `fbt` should integrate rather than reimplement.

## 7. Markdown AST and Content Transform

### unified / remark / rehype / retext

These are highly relevant for deterministic Markdown and content transformations.

`fbt` should treat remark-like transforms as runners/plugins and track their scripts/config as transform assets.

## 8. RAG and Document Pipeline Tools

### LlamaIndex / Haystack

Strong for indexing, retrieval, and RAG application workflows. They are application/framework layers, not artifact build control planes.

`fbt` can manage the corpus generation and evaluation artifacts that feed these systems.

## 9. Agent Workflow Tools

### LangGraph

Strong reference for stateful, tool-using agents. A LangGraph runner would be natural.

### CrewAI / AutoGen / Microsoft Agent Framework

Useful references for multi-agent orchestration. `fbt` should not absorb these concerns into core.

### OpenAI Agents SDK

Reference for tool use, guardrails, tracing, and model interaction.

### DSPy

Reference for optimizing LLM programs. It is closer to prompt/program optimization than file artifact build management.

## 10. Eval / Observability / Prompt Management

Langfuse, LangSmith, Braintrust, Promptfoo, MLflow GenAI, and Weave are important references for:

- prompt versioning
- eval datasets
- LLM judge workflows
- traces
- model metadata
- cost and token tracking

`fbt` should not replace these systems. It should provide local artifact lineage and support integration/export.

## 11. Lineage, Telemetry, and Protocols

### OpenLineage

Good export target for lineage. `fbt` should map artifact, transform, transform_run, eval, and policy metadata to OpenLineage facets where useful.

### OpenTelemetry

Good reference for traces and GenAI semantic conventions.

### MCP

MCP is relevant for exposing `fbt` project resources, artifacts, and tools to agents. It should be an integration layer, not the core transform protocol.

## 12. Security and Governance

Microsoft Presidio, Guardrails, and Promptfoo-style red teaming are references for validation, redaction, and guardrails.

For `fbt`, security should start with:

- scoped read/write paths
- tool allow/deny lists
- secret redaction
- cost/time limits
- review gates
- policy decisions recorded as runtime records

## 13. Opportunity for fbt

`fbt` can own a distinctive space:

- filesystem artifact graph
- LLM/agent transform contracts
- transform assets as graph nodes
- artifact versions and transform runs
- evals and approvals as part of build semantics
- local-first CLI with optional managed service
- export to OpenLineage / OpenTelemetry / MCP

## 14. Recommended Design Direction

1. Strongly watch DataChain as the closest adjacent product.
2. Do not reimplement Unstructured, Tika, Pandoc, or remark.
3. Design evals with lessons from Langfuse, Braintrust, and Promptfoo.
4. Use MCP and OpenLineage as integration/export paths.
5. Promise traceability, eval, and approval rather than full reproducibility.

## 15. References

- DataChain: https://datachain.ai/
- DVC: https://dvc.org/
- Snakemake: https://snakemake.github.io/
- Nextflow: https://www.nextflow.io/
- Dagster: https://dagster.io/
- Pachyderm: https://www.pachyderm.com/
- lakeFS: https://lakefs.io/
- Unstructured: https://unstructured.io/
- Apache Tika: https://tika.apache.org/
- Pandoc: https://pandoc.org/
- unified: https://unifiedjs.com/
- LlamaIndex: https://www.llamaindex.ai/
- Haystack: https://haystack.deepset.ai/
- LangGraph: https://www.langchain.com/langgraph
- OpenLineage: https://openlineage.io/
- OpenTelemetry: https://opentelemetry.io/
- Model Context Protocol: https://modelcontextprotocol.io/

