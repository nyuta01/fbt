# Agent Failures

This log records repeated or high-risk agent failure modes. Use it to turn
mistakes into deterministic guardrails.

Last reviewed: 2026-05-29.

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

## F-006 README over-explained before teaching the core model

- **Status**: `fixed`
- **Task**: `FBT-DOCS-UX-006`
- **Plan**: `docs/exec-plans/active/FBT-DOCS-UX-006-plan.md`

### Observation

README improvements kept adding more concrete detail, but the entry point still
forced first-time readers to assemble the product model from many features and
examples. The core user value was hidden behind breadth.

### Permanent fix

README now starts from one mental model:
`sources + instructions + runner -> artifact + build receipt`. The support
example now starts from the support lead's problem, shows that prior cases
contain the answer in a non-reusable form, then shows the generated manual,
receipt, review, and lineage before YAML or commands. Detailed capabilities
remain in linked docs instead of the README opening.

## F-007 README example made the workflow feel harder than the job

- **Status**: `fixed`
- **Task**: `FBT-DOCS-UX-007`
- **Plan**: `docs/exec-plans/active/FBT-DOCS-UX-007-plan.md`

### Observation

The support-manual README example still did not make the practical job obvious.
It used a scenario that did not immediately click, exposed transform details too
early, and listed commands without first saying what each one gives the user.

### Permanent fix

README now uses the simpler incident-notes to runbook workflow. The example
shows the existing incident evidence, the desired runbook, the small recipe
table, and then each command with its concrete result.

## F-008 README commands lacked visible execution feedback

- **Status**: `fixed`
- **Task**: `FBT-DOCS-UX-008`
- **Plan**: `docs/exec-plans/active/FBT-DOCS-UX-008-plan.md`

### Observation

Even after the example scenario became clearer, the README still made users
infer command value from prose. It did not show enough actual CLI feedback to
connect each lifecycle step with what appears in a terminal.

### Permanent fix

README now explains the example commands as checkpoints and includes partial
actual output: incident `plan` output plus shortened offline quickstart
lifecycle output for build, approval, and artifact history.

## F-009 Examples did not route users by intent

- **Status**: `fixed`
- **Task**: `FBT-EXAMPLES-UX-001`
- **Plan**: `docs/exec-plans/active/FBT-EXAMPLES-UX-001-plan.md`

### Observation

The examples were valid but did not make it obvious which one was safe to run
offline, which one represented the main practical value, what each command gave
the user, or why practical examples stopped without credentials. A mixed
`doctor` diagnostic also reported an executable runner as `error
RUNNER_COMMAND_OK` when a separate env var was missing.

### Permanent fix

`examples/README.md` now routes users by intent. Each example README explains
its job, inputs, outputs, credential boundary, command outcomes, and generated
receipts. Doctor now preserves per-diagnostic status so successful runner
checks remain `ok` even when another runner check fails.

## F-010 Source growth did not dirty glob-backed transforms

- **Status**: `fixed`
- **Task**: `FBT-EXAMPLES-UX-001`
- **Plan**: `docs/exec-plans/active/FBT-EXAMPLES-UX-001-plan.md`

### Observation

Testing the realistic daily-operation path showed that adding a new JSONL file
under a declared glob source could still leave the consuming transform skipped.
The manifest recorded resolved paths, but the source fingerprint only covered
the source definition.

### Permanent fix

Source fingerprints now include the resolved file set and per-file content
fingerprints. A new file under a declared glob or directory source causes
dependent transforms to plan as `run` with `source descriptor changed`.

## F-011 Examples implied JSONL and review as defaults

- **Status**: `fixed`
- **Task**: `FBT-EXAMPLES-UX-002`
- **Plan**: `docs/exec-plans/active/FBT-EXAMPLES-UX-002-plan.md`

### Observation

The practical examples and repeated-operation guidance could make users infer
that fbt's source model was centered on JSONL files and that review gates were
required for every artifact. That overfit the examples instead of teaching the
core model: fbt tracks declared filesystem sources, and review is a workflow
boundary, not a mandatory transform feature.

### Permanent fix

`examples/daily_qa_ops` now shows a daily workflow based on plain Markdown
directory sources, multiple source artifacts, and multiple outputs. fbt now has
no built-in review feature; approval and publishing belong in external
workflows.

## F-012 Daily example encoded a fixed source date

- **Status**: `fixed`
- **Task**: `FBT-EXAMPLES-UX-002`
- **Plan**: `docs/exec-plans/active/FBT-EXAMPLES-UX-002-plan.md`

### Observation

The first daily QA example used source paths like
`data/qa/2026-05-29/questions/`. That made the example impossible to read as
"run the same fbt flow once per day for newly arrived files" without editing
project config or creating new transforms for each date.

### Permanent fix

The example now uses stable processing-window source paths under
`data/qa/inbox/` and stable logical output paths under
`target/artifacts/.../latest/`. Daily scheduling and input-window preparation
remain outside fbt, while fbt records each run as artifact versions.

## F-013 Review leaked into the core product boundary

- **Status**: `fixed`
- **Task**: `FBT-UNIX-011`
- **Plan**: `docs/exec-plans/active/FBT-UNIX-011-plan.md`

### Observation

The Unix-style product boundary says fbt should build filesystem artifacts and
record receipts. The implementation and docs had grown a built-in review
command, approval state, review gates, review-related evals, and approval
facets. That made fbt look like a workflow approval system instead of a small
file build tool.

### Permanent fix

The review package, CLI command, approval state, review gates, `human_review`
eval type, config fields, example YAML, docs, smoke checks, conformance checks,
and standard-export approval facets were removed. The CLI smoke now asserts
that `fbt review` is an unknown command, so the removed feature cannot
accidentally reappear without changing tests.

## F-014 CLI ignored typos and empty selectors

- **Status**: `fixed`
- **Task**: `FBT-UNIX-012`
- **Plan**: `docs/exec-plans/active/FBT-UNIX-012-plan.md`

### Observation

The CLI accepted extra arguments for core commands. For example,
`fbt plan --bogus` still ran a plan, and `fbt build --select no_such` could
fall back to a broad build because an empty selector set was treated like no
selector. That violates the Unix-style expectation that scripts fail fast on
typos.

### Permanent fix

Command and subcommand argument validation now rejects unknown flags and extra
positionals. Selectors that match no transforms now fail. Build selection uses
the same selector semantics as plan, and CLI tests cover the typo cases.

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
