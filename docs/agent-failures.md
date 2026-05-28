# Agent Failures

This log records repeated or high-risk agent failure modes. Use it to turn
mistakes into deterministic guardrails.

Last reviewed: 2026-05-28.

No failures are currently active.

## F-001 Demo quickstart assumed repository cwd

- **Status**: `fixed`
- **Task**: `FBT-DOCS-UX-001`
- **Plan**: `docs/exec-plans/active/FBT-DOCS-UX-001-plan.md`

### Observation

Capturing the documented quickstart from a temporary directory revealed that
generated demo runner wrappers invoked bundled Go runner packages without
changing to the source checkout first. The docs claimed the quickstart worked
from a normal user project path, but the wrapper only worked from the repository
root.

### Permanent fix

Generated and checked-in demo wrappers now change to the source checkout before
running bundled demo runner packages. The knowledge-loop smoke builds an fbt
binary, runs the quickstart from a temporary directory, and includes `doctor`
so repository-cwd assumptions fail in verification.

## F-002 Ambiguous product diagram

- **Status**: `fixed`
- **Task**: `FBT-DOCS-UX-002`
- **Plan**: `docs/exec-plans/active/FBT-DOCS-UX-002-plan.md`

### Observation

The first support-loop graphic looked like an abstract architecture diagram but
was intended as a concrete quickstart explanation. It was not based on a
standard visualization or an actual product screenshot, and it did not state
what type of diagram it was, what claim it made, or how a user should read it.

### Permanent fix

The custom figure was removed from README and docs entry pages. Quickstart
behavior is now explained through CLI output, generated files, artifact
excerpts, and standard export commands. Future diagrams should be tied to a
standard visualization, an actual UI/output screenshot, or a clearly named
conceptual claim before publication.

## F-003 Quickstart scope ambiguity

- **Status**: `fixed`
- **Task**: `FBT-DOCS-UX-003`
- **Plan**: `docs/exec-plans/active/FBT-DOCS-UX-003-plan.md`

### Observation

The quickstart page showed real commands and outputs but did not say what the
support scenario was meant to represent. It could be read as a product demo,
model-quality benchmark, architecture explanation, or realistic manual
generation workflow.

### Permanent fix

Quickstart now opens by defining itself as a control-plane acceptance demo and
lists what each lifecycle stage proves. README, usage guide, and the
"What you can do today" page now say it is not a realistic manual-generation
workflow or model-quality benchmark.

## F-004 README did not answer the product question first

- **Status**: `fixed`
- **Task**: `FBT-DOCS-UX-004`
- **Plan**: `docs/exec-plans/active/FBT-DOCS-UX-004-plan.md`

### Observation

The README listed many accurate surfaces, commands, docs, examples, and
implementation details, but it still did not quickly answer what fbt is for,
what problem it solves, what it can be used for, and how a user should use it.

### Permanent fix

README was rewritten as a product entry point. It now starts with purpose and
use cases, explains the trust problem around generated operational artifacts,
uses the support resolution manual as the concrete example, and only then moves
to commands, project structure, boundaries, install, and reference docs.

## F-005 Concrete example lacked concrete content

- **Status**: `fixed`
- **Task**: `FBT-DOCS-UX-005`
- **Plan**: `docs/exec-plans/active/FBT-DOCS-UX-005-plan.md`

### Observation

The README concrete example named a support manual workflow but showed mostly
directory paths and commands. It did not show representative input records,
response-log content, transform logic, required output sections, or what fbt
records around the runner.

### Permanent fix

README now explains the example through actual source snippets, response-log
steps, transform YAML, required manual sections, output path, and fbt's
lineage/review responsibility before listing commands.

## Entry Template

```markdown
## F-001 Short title

- **Status**: `observing`
- **Task**: `FBT-H-002`
- **Plan**: `docs/exec-plans/active/example-plan.md`

### Observation

What happened.

### Permanent fix

What changed to make recurrence less likely.
```
