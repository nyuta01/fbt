# FBT-OPS-001 Document Daily High-Volume Source Operations Patterns

## Observation

Real users may add tens or hundreds of source files every day and run fbt on a
schedule. fbt should support this through simple composition, but it should not
grow into a scheduler, metadata database, or ingestion service.

## Decision

Document practical operations patterns that compose existing fbt primitives:
stable source directory conventions, date/window partitioning outside fbt,
selectors, external cron or CI, artifact partitioning, retention inspection,
and standard exports.

## Permanent Fix

Add a realistic daily-source workflow guide or example that shows how sources
and artifacts can both be plural and evolving. Keep the example runnable or
smoke-checked, and make clear which responsibilities belong to shell, cron,
CI, Git, object storage, or another system.

## Next Check

Run high-volume or daily-ops example smoke, docs-site build, and `make verify`.
