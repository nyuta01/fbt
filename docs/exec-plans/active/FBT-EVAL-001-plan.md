# FBT-EVAL-001 Add External Semantic And Evidence-Quality Check Examples

## Observation

The current eval story is strongest for deterministic checks such as required
sections. Real operational manuals also need evidence grounding, no-invention
checks, and usability checks. Those checks are important, but fbt core should
not become an LLM judge framework.

## Decision

Show how to compose fbt with external semantic, evidence-coverage, or grounding
checks while keeping judge/model logic outside core. fbt should record the
resulting eval or artifact receipt, not own the full quality engine.

## Permanent Fix

Add practical examples and docs for an external quality-check runner or command
transform that validates generated artifacts against source evidence. Keep the
boundary explicit: deterministic checks can remain simple; semantic checks are
external runners or downstream CI steps.

## Next Check

Run the external quality-check example smoke, docs scans for semantic eval
boundary language, docs-site build, and `make verify`.
