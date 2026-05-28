# FBT-REL-003 Publish MVP release artifacts and checksums

## Observation

Release artifact publication depends on `FBT-REL-002`: the public remote and
SSH signing setup are ready, and publication should start only after the signed
`v0.1.0` tag verifies locally and is pushed to GitHub.

## Decision

Do not publish or simulate a public MVP release before the signed tag exists.
Resume this task after the signed release baseline verifies.

## Permanent Fix

Kept this task separate from `FBT-REL-002` so release publication is not treated
as complete before artifacts and checksums are attached to the signed tag.

## Next Check

After `FBT-REL-002` is complete:

```sh
git remote -v
git tag -v v0.1.0
make dist-check
```
