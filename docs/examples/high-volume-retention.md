# High-Volume Retention Fixture

Status: MVP-ready
Updated: 2026-05-29
Audience: teams running fbt repeatedly over growing source sets

## Purpose

fbt keeps local history by default:

```text
.fbt/state/      run receipts, manifests, current pointers
.fbt/artifacts/  immutable artifact version snapshots
```

This is the MVP retention policy:

```text
keep_all
```

There is no automatic cleanup and no destructive prune command in MVP. The
supported operation is inspection plus external archival.

## Fixture

The high-volume smoke creates a temporary project, adds eight source batches,
builds the same artifact after each batch, and inspects retention:

```sh
make retention-high-volume-smoke
```

Expected result:

```text
retention-high-volume-smoke: ok
```

The smoke validates this report shape:

```text
Artifact retention
  Policy               keep_all
  Archive unit         .fbt/state + .fbt/artifacts
  Artifact versions    8
  Current versions     1
  Historical versions  7
  Protected versions   1 current pointer(s)
  Prune                not supported in MVP; future prune must dry-run first
  Action               no files removed; archive state and artifact dirs together
```

It also checks `--json` output and verifies that `archive_roots` includes both
`.fbt/state` and `.fbt/artifacts`. The JSON report also includes
`archive_unit: state_and_artifacts`, `protected_version_ids`,
`prune_supported: false`, and `dry_run_required: true`.

## Operational Guidance

For daily or high-volume projects:

1. Let fbt keep all local versions while the project is active.
2. Use `fbt artifact retention --json` in CI or scheduled checks to monitor
   growth.
3. Archive `.fbt/state/` and `.fbt/artifacts/` together when moving history to
   backup, object storage, or CI artifacts.
4. Do not delete immutable artifact directories without the matching state
   records unless you intentionally accept broken historical pointers.

For production daily runs, archive the run bundle with those roots:

```text
.fbt/state/
.fbt/artifacts/
target/ops/runs/<run-id>/
```

`examples/daily_qa_ops/ops/archive-fbt-evidence.sh` writes
`target/ops/archives/<run-id>/fbt-evidence.tar.gz` plus an
`archive-manifest.json` describing that restore unit. Store that archive as a
CI artifact or in external object storage before applying any external
lifecycle policy. fbt's base tool still reports retention and archive
boundaries; it does not prune historical versions automatically.

The current command is read-only:

```sh
fbt artifact retention --project-dir my_project
```

Use external tools for archive windows:

```sh
tar -czf fbt-history-2026-05-29.tgz my_project/.fbt/state my_project/.fbt/artifacts
rsync -a my_project/.fbt/state my_project/.fbt/artifacts backup:/fbt/my_project/
```

Future destructive cleanup, if added, must be explicit, dry-run-first,
receipt-aware, current-pointer-protecting, and conformance-covered.
