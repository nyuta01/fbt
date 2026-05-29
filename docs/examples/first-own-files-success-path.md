# First Own-Files Success Path

Status: MVP-ready
Updated: 2026-05-29
Audience: first-time users turning their own directory of files into one
versioned artifact

## Goal

This is the smallest practical path after the offline quickstart. It answers:

> I have my own files in a folder. How do I get the first fbt artifact and
> receipt without designing a whole project?

Use the support template as scaffolding, replace the sample input files, then
build one artifact.

## 1. Create The Scaffold

```sh
fbt init my_support_manual --template support
```

The generated project already has the pieces fbt needs:

| Role | Generated file | What you replace first |
|---|---|---|
| Source declaration | `sources/support.yml` | Usually keep the path and copy your files into it. |
| Source files | `data/support/tickets/*.jsonl` | Replace with your own records. |
| Instruction | `assets/support_style_guide.md` | Replace with your team guidance. |
| Runner | `fs_project.yml` | Keep `demo.llm` for local proof, then switch to an external runner. |
| Artifact recipe | `transforms/support/case_summaries.yml` | Keep for first proof; adjust once the loop works. |

## 2. Put Your Files Under The Declared Source Path

For the first loop, do not redesign the project. Replace the sample ticket file
with two of your own JSONL files:

```sh
rm my_support_manual/data/support/tickets/*.jsonl

cat >my_support_manual/data/support/tickets/2026-05-29-login.jsonl <<'JSONL'
{"id":"MY-101","summary":"Admin cannot receive password reset email after company domain change","impact":"Workspace admin blocked from console","resolution_status":"resolved"}
JSONL

cat >my_support_manual/data/support/tickets/2026-05-29-billing.jsonl <<'JSONL'
{"id":"MY-102","summary":"Customer asks why seat count changed after SSO group sync","impact":"Unexpected invoice estimate","resolution_status":"resolved"}
JSONL
```

This works because `sources/support.yml` declares:

```yaml
sources:
  - name: support
    artifacts:
      - name: raw_tickets
        type: jsonl_directory
        path: data/support/tickets/*.jsonl
```

fbt fingerprints the resolved file set and contents. New, removed, or edited
files under that glob make dependent artifacts dirty.

## 3. Replace The Instruction

```sh
cat >my_support_manual/assets/support_style_guide.md <<'MD'
# Support Style Guide

- Separate facts from assumptions.
- Include the customer impact and next action.
- Keep generated notes short enough to review in a pull request.
MD
```

The transform already references this asset:

```yaml
assets:
  - ref: support_style_guide
```

## 4. Prove The Loop Locally

The template uses deterministic demo runners so the first proof needs no
provider account.

```sh
fbt doctor --project-dir my_support_manual
fbt plan --project-dir my_support_manual --select case_summaries
fbt build --project-dir my_support_manual --select case_summaries
fbt artifact explain case_summaries --project-dir my_support_manual
```

Expected `plan` shape:

```text
Plan
  selected  1
  run       1

RUN     case_summaries
        because  no previous successful run
        because  output missing
        output   case_summaries
```

The artifact appears at:

```text
my_support_manual/target/artifacts/support/case_summaries/index.md
```

`artifact explain` is the proof that your own files and instruction asset are
part of the build receipt:

```text
Artifact: case_summaries

Inputs
  ok input   support.raw_tickets     path=data/support/tickets/*.jsonl
  ok asset   support_style_guide     path=assets/support_style_guide.md
  ok runner  demo.llm

Outputs
  case_summaries  target/artifacts/support/case_summaries
```

## 5. Replace The Runner When The Local Loop Works

The local proof uses `demo.llm`. For real generation, replace the runner in
`fs_project.yml` with an external command that speaks the fbt runner protocol:

```yaml
runners:
  - name: openai.responses
    type: llm
    protocol: stdio_jsonrpc
    command: fbt-runner-openai
    args: ["responses"]
    env:
      - OPENAI_API_KEY
```

Then change the transform runner:

```yaml
runner: openai.responses
```

Run `fbt doctor` again. If credentials or commands are missing, fbt fails before
calling the runner and tells you which environment variable or command to fix.

## 6. Add Another File Tomorrow

```sh
cat >my_support_manual/data/support/tickets/2026-05-30-export.jsonl <<'JSONL'
{"id":"MY-103","summary":"Customer asks whether scheduled exports use UTC or workspace timezone","impact":"Admin unsure how to communicate report timing","resolution_status":"open"}
JSONL

fbt plan --project-dir my_support_manual --select case_summaries
```

Expected reason:

```text
RUN     case_summaries
        because  source descriptor changed
```

That is the core fbt loop: keep adding or changing source files, preview the
dirty artifact, build a new version, and inspect the receipt.
