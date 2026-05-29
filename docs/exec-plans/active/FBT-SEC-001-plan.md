# FBT-SEC-001 Document OS Sandbox Execution Profiles For High-Security Use

## Observation

fbt policy, path containment, output boundaries, and adapter safety checks are
covered, but OS-level sandboxing is intentionally external to core. Users with
strict security requirements need concrete execution profiles.

## Decision

Document how to run fbt and runner commands inside external isolation layers
such as CI sandboxes, containers, macOS sandboxing, Linux namespace/seccomp
profiles, or network-denied environments.

## Permanent Fix

Add a security guide that keeps the core boundary clear: fbt records and
enforces its own file/build contract, while OS-level process and network
isolation belongs to the execution environment.

## Next Check

Run docs scans for sandbox boundary language and `make verify`.
