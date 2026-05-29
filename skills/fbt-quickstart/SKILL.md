---
name: fbt-quickstart
description: >-
  Install and run the standard fbt CLI quickstart: create the offline support
  template, check readiness, plan, build, inspect artifacts, and explain the
  receipt. Invoke when the user wants to try fbt, verify the CLI works, or see
  the basic file-build loop before wiring a real runner.
---

# fbt quickstart

Use fbt's deterministic support template to prove the CLI is installed and the
local file-build loop works before introducing OpenAI, Claude Code, Codex, or
another external runner.

The core model is:

```text
source files + instructions + external runner
  -> generated artifact
  -> local build receipt, version, diff, lineage, and standard exports
```

In this quickstart, the source files are support ticket examples, the
instructions are bundled prompt/style assets, the runner is a deterministic
demo command, and the artifacts are Markdown support summaries. fbt itself does
not write the content directly; it decides what should run, calls the runner,
checks the output, commits an immutable artifact version, and records why.

## When this skill applies

- The user asks to install or try the `fbt` CLI.
- The user wants a first successful local run with no provider credentials.
- The user asks what fbt does in practice or how to inspect the generated
  artifact receipt.

This skill does not apply when the user already has a failing real runner. Use
`fbt doctor`, `fbt plan`, and the project-specific runner docs to debug that
case after the quickstart works.

## Procedure

1. **Install fbt if needed.**

   ```bash
   command -v fbt >/dev/null || \
     curl -fsSL https://raw.githubusercontent.com/nyuta01/fbt/main/install.sh | sh
   fbt version
   ```

   If the installer says `$HOME/.local/bin` is not on `PATH`, add it for the
   current shell before continuing.

2. **Create the offline project.**

   ```bash
   fbt init knowledge_ops --template support
   ```

   This creates a local project with source files, prompts/assets, policies,
   deterministic demo runners, and transform declarations.

   The important files are:

   ```text
   knowledge_ops/data/support/tickets/       source files
   knowledge_ops/assets/                     instructions and style guide
   knowledge_ops/transforms/support/         artifact recipes
   knowledge_ops/bin/fbt-demo-llm-runner     external demo runner
   ```

3. **Check readiness.**

   ```bash
   fbt doctor --project-dir knowledge_ops
   ```

   This answers: "Can this project run here?" Continue only when doctor reports
   no errors. The demo template does not need provider credentials.

4. **Preview what will run.**

   ```bash
   fbt plan --project-dir knowledge_ops --select tag:support
   ```

   This answers: "What would fbt regenerate, skip, or block before any runner
   is called?" Expected shape:

   ```text
   Plan
     selected  2
     run       2
     blocked   0
   ```

5. **Build the support artifacts.**

   ```bash
   fbt build --project-dir knowledge_ops --select tag:support
   ```

   This calls the external demo runner and commits versioned artifacts.
   Expected result includes `SUCCESS case_summaries` and
   `SUCCESS weekly_support_insights`.

   The generated files land under:

   ```text
   knowledge_ops/target/artifacts/support/case_summaries/index.md
   knowledge_ops/target/artifacts/support/weekly_insights.md
   knowledge_ops/.fbt/state/
   knowledge_ops/.fbt/artifacts/
   ```

6. **Inspect the current artifact and receipt.**

   ```bash
   fbt artifact show case_summaries --project-dir knowledge_ops
   fbt artifact explain case_summaries --project-dir knowledge_ops
   ```

   `show` answers where the current artifact lives and which version is
   current. `explain` answers which sources/assets/runner/evals and build
   decision produced the receipt.

7. **Export lineage when needed.**

   ```bash
   fbt export openlineage --project-dir knowledge_ops \
     --output knowledge_ops/target/lineage/openlineage.ndjson
   ```

   Use this when the user wants to connect fbt output to standard lineage or
   metadata tools.

## Verify

```bash
test -f knowledge_ops/target/artifacts/support/case_summaries/index.md
fbt artifact show case_summaries --project-dir knowledge_ops
fbt artifact explain case_summaries --project-dir knowledge_ops
```

All commands should exit 0. The artifact should have a current
`case_summaries@sha256:...` version.

If `plan` says `skipped` after a successful build, that is expected: fbt has a
receipt proving the current sources and instructions already produced the
current artifact version.

## Next steps

- Replace the template source files and assets with the user's real files, then
  repeat `plan`, `build`, and `artifact explain`.
- Keep deterministic demo runners until the file graph and artifact contract
  are correct.
- Switch the runner command only after the local loop is understood; fbt core
  does not include provider SDKs or agent runtimes.
