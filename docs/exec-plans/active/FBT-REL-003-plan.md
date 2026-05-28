# FBT-REL-003 Publish MVP release artifacts and checksums

## Observation

The signed `v0.1.0` tag verifies locally and is pushed to GitHub. The release
needs downloadable CLI artifacts and checksums attached to that tag so users can
install or inspect the MVP without building from source.

## Decision

Publish a GitHub release for `v0.1.0` from the signed tag, attach
cross-platform CLI archives and a `SHA256SUMS` file, and keep the generated
artifacts outside the repository because `dist/` is ignored build output.

## Permanent Fix

Created the public release at
`https://github.com/nyuta01/fbt/releases/tag/v0.1.0` with these assets:

- `fbt_0.1.0_darwin_amd64.tar.gz`
- `fbt_0.1.0_darwin_arm64.tar.gz`
- `fbt_0.1.0_linux_amd64.tar.gz`
- `fbt_0.1.0_linux_arm64.tar.gz`
- `fbt_0.1.0_windows_amd64.zip`
- `fbt_0.1.0_windows_arm64.zip`
- `SHA256SUMS`
- `version-darwin-arm64.json`

The tag-triggered GitHub `verify` workflow completed successfully for both
`main` and `v0.1.0`.

## Next Check

Release integrity checks:

```sh
git remote -v
git tag -v v0.1.0
make dist-check
shasum -a 256 -c dist/release/v0.1.0/SHA256SUMS
gh release view v0.1.0 --repo nyuta01/fbt
gh run list --repo nyuta01/fbt --limit 5
```
