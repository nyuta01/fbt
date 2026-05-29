<p align="center">
  <img src="https://raw.githubusercontent.com/nyuta01/fbt/main/apps/docs/public/favicon.svg" alt="fbt" width="96" height="96" />
</p>

# fbt Agent Skills

A distributable pack of `SKILL.md` files that teach AI coding agents how to use
the `fbt` CLI. Compatible with the `skills.sh` ecosystem and with the TanStack
Intent registry.

This pack ships product knowledge about fbt itself: how to install the CLI,
run the offline quickstart, inspect generated artifacts, and know where fbt's
responsibility ends. It is not a runner, provider SDK, scheduler, or project
template.

## Installing

### Via skills.sh

```bash
# Project-scoped install
npx skills add nyuta01/fbt

# Global install
npx skills add -g nyuta01/fbt

# Install just the standard quickstart skill
npx skills add nyuta01/fbt --skill fbt-quickstart
```

The CLI symlinks the skill files into your agent runtime's discovery path.

### Via npm

```bash
npm install --save-dev fbt-agent-skills
```

The npm package ships the same `SKILL.md` file and is tagged with the
`tanstack-intent` keyword for registry discovery.

### Manual

```bash
git clone https://github.com/nyuta01/fbt.git /tmp/fbt
cp -r /tmp/fbt/skills/fbt-quickstart .claude/skills/
```

## What's Included

| Skill | When to invoke |
|---|---|
| `fbt-quickstart` | Install fbt if needed, run the offline support template, build artifacts, and inspect receipts. |

## SKILL.md Format

Every skill is a directory containing exactly one `SKILL.md`:

```text
skills/
└── fbt-quickstart/
    └── SKILL.md
```

Frontmatter follows the universal format used by Claude Skills, skills.sh, and
TanStack Intent:

```yaml
---
name: fbt-quickstart
description: >-
  One-line summary used by agents to decide whether the skill applies.
---
```

The body is plain Markdown. fbt skills follow these conventions:

1. Lead with a one-line summary.
2. Include a "When this skill applies" section.
3. Provide a numbered procedure with concrete CLI commands.
4. End with a "Verify" section the agent can run.

## License

Apache-2.0, same as fbt. See the repository root.
