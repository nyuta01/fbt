# FBT-DOCSITE-001 Add Folio-inspired documentation site

## Observation

The repository had strong Markdown source-of-truth docs, but no navigable
published documentation site comparable to Folio's Astro/Starlight docs at
`https://nyuta01.github.io/folio/`.

## Decision

Add `apps/docs` as a lightweight Astro/Starlight site that mirrors the Folio
docs information architecture: landing page, introduction, get started, CLI,
runners, lineage/standards, and reference sections. Keep source-of-truth specs
in `docs/` and use the site as the user-facing entry surface.

## Permanent Fix

Added an Astro/Starlight docs app, branded assets, curated MVP content, GitHub
Pages workflow, and `make docs-site-build` wired into `make verify` so the site
does not drift silently. Hardened the task-state harness so done tasks cannot
pass locally by referencing ignored, generated paths that are absent in CI.

## Next Check

```sh
make docs-site-build
make verify
```
