# FBT-POLICY-001 Enforce output-size limits for directory artifacts

## Observation

`max_output_bytes` is enforced only when an artifact descriptor has `Size`.
Directory descriptors currently record `FileCount` but leave `Size` empty, so a
runner can produce a large `directory` or `markdown_directory` candidate and
pass the policy check.

## Decision

Make directory descriptors carry aggregate byte size, and make policy checks
apply the same size limit to file and directory artifact types. Preserve the
existing digest and file-count semantics.

## Permanent Fix

Directory descriptors now carry aggregate byte size while preserving file-count
and digest semantics. Policy and build regressions cover directory artifacts
that exceed `max_output_bytes`, including the invariant that denied outputs do
not update current artifact pointers or immutable versions.

## Next Check

Done. Targeted artifact, policy, and build tests pass, and `make verify` passes.
