# FBT-DOCS-UX-013 Strengthen runner and standards docs with runnable examples

## Observation

The runner and standards docs are accurate but thinner than README. Pages such
as external runners, authoring contract, OpenAI runner, lineage model, and
project config describe the concepts, but they do not always include enough
runnable commands, minimal config, expected output, and concrete file pointers
to stand alone for a first-time user.

Updated observation: the thin pages now carry runnable command examples and
short output excerpts. Runner pages show project YAML, conformance commands,
missing-env doctor diagnostics, and smoke matrix usage. Standards pages show a
local fixture, artifact explain output, export commands, and export summaries.
Project config shows the concrete `fs_project.yml` / `sources` / `transforms`
shape behind the incident runbook example.

## Decision

Keep the docs site concise, but make each thin page self-contained enough for a
user to execute or verify the concept. Runner pages should include install/use
patterns, conformance checks, and failure diagnostics. Standards pages should
show exact export commands, expected summaries, and concrete backend handoff
shape.

## Permanent Fix

Add runnable examples and short output excerpts to the docs-site runner,
lineage/standards, and project-config pages. Cross-link to the deeper repository
specs only after the page has answered the practical "how do I use this?"
question.

Implemented in:

- `apps/docs/src/content/docs/runners/external-runners.mdx`
- `apps/docs/src/content/docs/runners/openai-runner.mdx`
- `apps/docs/src/content/docs/runners/authoring-contract.mdx`
- `apps/docs/src/content/docs/standards/lineage-model.mdx`
- `apps/docs/src/content/docs/standards/openlineage.mdx`
- `apps/docs/src/content/docs/standards/opentelemetry.mdx`
- `apps/docs/src/content/docs/standards/visualization.mdx`
- `apps/docs/src/content/docs/reference/project-config.mdx`
- `docs/runner-adapters.md`
- `docs/runner-authoring-guide.md`
- `docs/standard-visualization-guide.md`

## Next Check

Run docs-site build and `make verify`; add deterministic docs scans if the new
examples introduce phrases that should remain guarded.

Completed:

- runner/docs scan for conformance commands, fake OpenAI adapter mode, doctor
  diagnostics, project config examples, and smoke matrix examples
- standard export docs scan for OpenLineage and OTel summary output
- `npm --prefix apps/docs run build`
- `make verify`
