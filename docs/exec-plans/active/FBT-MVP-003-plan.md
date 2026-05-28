# FBT-MVP-003 Implement artifact descriptors and safe path handling

## Observation

The manifest can now represent logical artifacts and graph edges, but there is
no implementation for authoritative file/directory descriptors, artifact version
IDs, or reusable path-safety checks. Build and state work need these primitives
before runner output can be trusted.

## Decision

Implement the descriptor and path baseline:

- compute SHA-256 file descriptors from exact bytes
- compute canonical directory descriptors from sorted relative POSIX paths,
  file sizes, and file digests
- reject symlinks and path escapes during descriptor computation
- map YAML artifact aliases to descriptor artifact type IDs
- generate full digest artifact version IDs
- expose reusable project-relative path containment helpers

## Permanent Fix

Added artifact and security tests for exact byte file digests, stable canonical
directory digests, artifact version IDs, project-relative path escape rejection,
containment checks, and symlink rejection.

## Next Check

Run:

```sh
make verify
```

Result on 2026-05-28: pass.
