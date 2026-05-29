# FBT-REL-007 Standardize Core Release Automation

## Observation

The `v0.2.1` release proved that fbt can publish usable core CLI archives, but
the operation still depended on manual version-drift updates, manual asset
upload, and a human waiting for CI after publishing. The first `v0.2.0` attempt
also showed that tag publication could happen before a clean-checkout release
gate caught all issues.

## Decision

Use the common GitHub Releases shape as the default path: a maintainer prepares
a release candidate commit, runs a local preflight, creates a signed annotated
root `vX.Y.Z` tag, and pushes it. A GitHub Actions workflow then verifies the
tagged commit, builds archives, checks `SHA256SUMS`, and publishes the release.
Do not make GitHub artifact attestations part of the required baseline yet;
keep them as a possible additive supply-chain improvement.

## Permanent Fix

Add a release version drift guard, a local `release-preflight` command, and a
tag-triggered `.github/workflows/release-core.yml` workflow. Keep tag CI in the
release workflow so `verify.yml` does not duplicate tag checks. Update release
docs, task state, quality notes, and the failure log so the clean-checkout
release gap is documented as a fixed process failure.

## Next Check

Release automation checks:

```sh
make release-version-check
scripts/release-preflight.sh --allow-dirty --allow-existing-tag --skip-verify v0.2.1
make verify
```
