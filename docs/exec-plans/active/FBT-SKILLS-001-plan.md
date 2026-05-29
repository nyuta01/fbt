# FBT-SKILLS-001 Add Standard Agent Skill Pack

## Observation

Agents can install and run the fbt CLI, but there was no skills.sh-compatible
entry point that teaches an agent the standard fbt loop. Folio ships a
product-level `skills/` pack for this purpose; fbt needed the same pattern, but
only one standard skill should be introduced for now.

## Decision

Add a product-level agent skill pack under `skills/` with exactly one standard
skill: `fbt-quickstart`. Do not add project-local skill commands to the fbt CLI
yet. The first step is a distributable `SKILL.md` that external agent runtimes
can install through skills.sh or npm.

## Permanent Fix

Add `skills/fbt-quickstart/SKILL.md`, skills.sh/TanStack Intent package
metadata, a concise skills README, and a deterministic `agent-skills-check`
behind `make verify`. Link the skill pack from README without expanding the
public CLI surface.

## Next Check

Agent skills checks:

```sh
make agent-skills-check
make verify
```
