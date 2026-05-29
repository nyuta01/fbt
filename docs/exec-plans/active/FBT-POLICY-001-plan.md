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

Add descriptor and policy tests for a directory artifact whose total byte count
exceeds `max_output_bytes`. Add a build-level regression so denied directory
outputs do not advance current artifact pointers or immutable versions.

## Next Check

Run targeted artifact/policy/build tests, then `make verify`.
