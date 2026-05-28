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
