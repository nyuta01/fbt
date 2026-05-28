# FBT-REL-002 Prepare public repository remote and signed release baseline

## Observation

The repository now has `origin` configured for `github.com:nyuta01/fbt`, and a
non-interactive `git ls-remote --heads origin` check succeeds. It still has no
local commit signing settings, no signed release tag, and the latest local
commit is unsigned. Those remaining steps require maintainer signing credentials
and a release-history decision.

## Decision

Keep the current local history intact and document the maintainer-owned release
baseline: keep the public GitHub remote, enable signing from the release point
forward unless the maintainer intentionally rewrites history, push `main`, and
create a signed `v0.1.0` tag after `make verify` passes.

## Permanent Fix

Added maintainer release/signing guidance to `CONTRIBUTING.md`, release
integrity expectations to `SECURITY.md`, and a tag trigger to the existing
GitHub `verify` workflow so pushed release tags run `make verify`.

## Next Check

Maintainer-owned checks:

```sh
git remote -v
git ls-remote --heads origin
git config --get commit.gpgsign
git log --show-signature -1
git tag -v v0.1.0
make verify
```
