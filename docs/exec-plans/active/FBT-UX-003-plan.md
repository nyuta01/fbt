# FBT-UX-003 Add artifact path, show, and history commands

## Observation

Generated artifacts were visible in state files and `artifact versions`, but
users needed to piece together logical output paths, immutable storage paths,
runner/model metadata, confidence, and generating run IDs manually.

## Decision

Keep artifact inspection in the existing `fbt artifact` namespace. Add a
scriptable path command, enrich `show`, and add `history` for version lists
without changing the internal state model.

## Permanent Fix

Implemented `fbt artifact path TARGET`, enriched `fbt artifact show TARGET`,
and added `fbt artifact history TARGET`. The commands include artifact version,
logical path, immutable storage path, digest, artifact type, runner/model,
confidence, generating run, commit time, and materials where available. Added
CLI tests, knowledge-loop smoke coverage, and documentation.

## Next Check

Run:

```sh
make verify
```
