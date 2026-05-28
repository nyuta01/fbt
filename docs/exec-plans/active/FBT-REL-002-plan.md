# FBT-REL-002 Prepare public repository remote and signed release baseline

## Observation

The repository now has `origin` configured for `github.com:nyuta01/fbt`, and a
non-interactive `git ls-remote --heads origin` check succeeds. SSH signing is
configured locally for the release point using the maintainer SSH key loaded in
the agent. Earlier commits remain unsigned by policy; signing starts from the
release-baseline commit and tag. `main` and signed tag `v0.1.0` have been
pushed to GitHub.

## Decision

Keep the current local history intact, use SSH signing from the release point
forward, push `main`, and create a signed `v0.1.0` tag after `make verify`
passes.

## Permanent Fix

Added maintainer release/signing guidance to `CONTRIBUTING.md`, release
integrity expectations to `SECURITY.md`, a tag trigger to the existing GitHub
`verify` workflow, and local SSH signing configuration so release-baseline
commits and tags can be signed without rewriting existing history.

## Next Check

Release-baseline checks:

```sh
git remote -v
git ls-remote --heads origin
git config --get commit.gpgsign
git log --show-signature -1
git tag -v v0.1.0
make verify
```
