# FBT-UX-004 Add review show and safer approval guidance

## Observation

Users could approve an artifact version from `review status`, but there was no
single review surface showing the selected version, output paths, digest,
runner/model, generating run, and inspection commands before promotion.

## Decision

Add `fbt review show TARGET` as the review surface and make pending
`review status` point to it. Keep approval explicit, but show the approve and
reject commands only after artifact inspection commands.

## Permanent Fix

Implemented `fbt review show TARGET [--version VERSION_ID]` with artifact
metadata, immutable storage path, runner/model, generating run, optional diff
guidance, and approve/reject-after-review commands. Added pending status
guidance, CLI tests, knowledge-loop smoke coverage, and documentation.

## Next Check

Run:

```sh
make verify
```
