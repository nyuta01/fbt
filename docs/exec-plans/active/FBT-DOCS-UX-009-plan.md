# FBT-DOCS-UX-009 Rework README first-user journey

## Observation

The README described fbt's value, but a first-time reader still had to infer
the order of understanding: concept, project structure, first run, practical
workflow, and deeper docs. That made the tool feel clearer than before but not
yet self-explanatory from README alone.

## Decision

Reworked README as the primary first-user path: value first, then command
questions, project anatomy, an offline successful loop, a real workflow example,
tool boundaries, and goal-oriented next links.

## Permanent Fix

Kept the README within the 220-line docs guard and preserved required source of
truth links so future edits cannot silently turn it into a sprawling manual.

## Next Check

Done. `validate_docs` and `make verify` pass.
