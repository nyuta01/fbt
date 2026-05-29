# FBT-STATE-003 Validate Retention Guidance With A High-Volume Fixture

## Observation

`fbt artifact retention` gives a safe read-only answer for local state growth,
and `keep_all` is the right MVP default. The remaining concern is whether
users running daily high-volume projects can understand growth and archive
boundaries from concrete evidence rather than policy text alone.

## Decision

Validate retention guidance with a fixture that creates many source changes or
artifact versions and then inspects state/artifact growth. Keep cleanup
non-destructive unless a future task explicitly designs receipt-aware pruning.

## Permanent Fix

Add a high-volume fixture or smoke that exercises retention inspection under
many versions, then document what to archive together and what fbt intentionally
does not delete in MVP.

## Next Check

Run the high-volume retention fixture, targeted state/CLI tests, docs scans,
and `make verify`.
