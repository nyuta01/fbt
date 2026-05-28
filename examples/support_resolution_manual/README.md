# Support Resolution Manual Example

This is the support-ops practical example.

It turns tickets and support response notes into a support manual:

```text
customer tickets + agent response logs + product docs + approved macros
  -> target/artifacts/support/support_resolution_manual.md
  -> .fbt receipt with sources, runner, checks, version, and lineage
```

If you are new to fbt, read the incident runbook example first. This example is
equally realistic, but it has more input types.

## Run The Workflow

Preview the work:

```sh
fbt plan --project-dir examples/support_resolution_manual --select support_resolution_manual
```

After setting credentials, build the manual:

```sh
export OPENAI_API_KEY=...
fbt build --project-dir examples/support_resolution_manual --select support_resolution_manual
```

You get:

```text
target/artifacts/support/support_resolution_manual.md
.fbt/artifacts/<artifact_version>/content
.fbt/state/artifact_versions.json
.fbt/state/run_results.jsonl
```

Inspect where it came from:

```sh
fbt artifact show support_resolution_manual --project-dir examples/support_resolution_manual
fbt artifact history support_resolution_manual --project-dir examples/support_resolution_manual
fbt artifact explain support_resolution_manual --project-dir examples/support_resolution_manual
```

Generate local docs:

```sh
fbt docs generate --project-dir examples/support_resolution_manual
```

Human approval and publishing should happen in Git, PR, CI, release, or catalog
workflows outside fbt.
