# FBT-REL-006 Publish Current Core CLI Release

## Observation

GitHub Releases still published `v0.1.0` as the latest core CLI, while `main`
contained more than one hundred commits of product, CLI, runner, docs, and
state improvements after that tag. The first `v0.2.0` release attempt exposed
that `examples/data_tool_interop` relied on locally ignored dbt `target/`
fixture files, so a clean checkout could not run the practical example smoke.

## Decision

Cut a new core CLI release as `v0.2.1`. This is a core CLI release only:
official runner adapters remain separate Go module installs and do not receive
module-scoped tags in this task.

## Permanent Fix

Promote the source default version and public install docs to `0.2.1`, add a
deterministic cross-platform core CLI release asset script, explicitly track
the dbt example fixture needed by clean checkout CI, run the verification gate,
create a signed `v0.2.1` tag, publish GitHub release archives with `SHA256SUMS`,
and verify the release after upload.

## Next Check

Release integrity checks:

```sh
make verify
scripts/release-core-cli.sh v0.2.1
shasum -a 256 -c dist/release/v0.2.1/SHA256SUMS
git tag -v v0.2.1
gh release view v0.2.1 --repo nyuta01/fbt
```
