# FBT-STATE-003 Validate Retention Guidance With A High-Volume Fixture

## Observation

`fbt artifact retention` gives a safe read-only answer for local state growth,
and `keep_all` is the right MVP default. The remaining concern is whether
users running daily high-volume projects can understand growth and archive
boundaries from concrete evidence rather than policy text alone.

Updated observation: retention guidance now has fixture-backed evidence.
`retention-high-volume-smoke` creates eight artifact versions, checks current
and historical counts, verifies JSON archive roots, and confirms the command
removes no files.

## Decision

Validate retention guidance with a fixture that creates many source changes or
artifact versions and then inspects state/artifact growth. Keep cleanup
non-destructive unless a future task explicitly designs receipt-aware pruning.

## Permanent Fix

Add a high-volume fixture or smoke that exercises retention inspection under
many versions, then document what to archive together and what fbt intentionally
does not delete in MVP.

Implemented in:

- `scripts/smoke-retention-high-volume.sh`
- `Makefile`
- `docs/examples/high-volume-retention.md`
- `docs/usage-guide.md`
- `docs/state-and-run-results-spec.md`
- `apps/docs/src/content/docs/cli/inspection.mdx`

## Next Check

Run the high-volume retention fixture, targeted state/CLI tests, docs scans,
and `make verify`.

Completed:

- `make retention-high-volume-smoke`
- `go test ./internal/state ./internal/cli`
- retention docs scan for `keep_all`, archive roots, read-only behavior, and
  high-volume fixture guidance
- `make verify`
