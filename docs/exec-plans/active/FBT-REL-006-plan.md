# FBT-REL-006 Publish Current Core CLI Release

## Observation

GitHub Releases still published `v0.1.0` as the latest core CLI, while `main`
contained more than one hundred commits of product, CLI, runner, docs, and
state improvements after that tag.

## Decision

Cut a new core CLI release as `v0.2.0`. This is a core CLI release only:
official runner adapters remain separate Go module installs and do not receive
module-scoped tags in this task.

## Permanent Fix

Promote the source default version and public install docs to `0.2.0`, add a
deterministic cross-platform core CLI release asset script, run the verification
gate, create a signed `v0.2.0` tag, publish GitHub release archives with
`SHA256SUMS`, and verify the release after upload.

## Next Check

Release integrity checks:

```sh
make verify
scripts/release-core-cli.sh v0.2.0
shasum -a 256 -c dist/release/v0.2.0/SHA256SUMS
git tag -v v0.2.0
gh release view v0.2.0 --repo nyuta01/fbt
```
