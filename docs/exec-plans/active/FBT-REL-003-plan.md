# FBT-REL-003 Publish MVP release artifacts and checksums

## Observation

Release artifact publication depends on `FBT-REL-002`: a public remote, a
maintainer signing setup, and a signed `v0.1.0` tag. Those prerequisites are
not present locally, so there is no trusted release target for attaching
artifacts or checksums.

## Decision

Do not publish or simulate a public MVP release from an unsigned local-only
repository. Resume this task after the maintainer completes the signed release
baseline.

## Permanent Fix

Marked this task blocked on `FBT-REL-002` so release publication is not treated
as agent-complete before a signed tag and remote release target exist.

## Next Check

After `FBT-REL-002` is complete:

```sh
git remote -v
git tag -v v0.1.0
make dist-check
```
