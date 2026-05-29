#!/usr/bin/env python3
"""Validate the distributable fbt agent skills pack."""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
SKILLS = ROOT / "skills"

errors: list[str] = []


def fail(message: str) -> None:
    errors.append(message)


def read_text(path: Path) -> str:
    return path.read_text(encoding="utf-8")


required_files = [
    SKILLS / "README.md",
    SKILLS / "LICENSE",
    SKILLS / "package.json",
    SKILLS / ".claude-plugin" / "marketplace.json",
    SKILLS / "fbt-quickstart" / "SKILL.md",
]

for path in required_files:
    if not path.exists():
        fail(f"missing required skills file: {path.relative_to(ROOT)}")

skill_dirs = sorted(
    path.name
    for path in SKILLS.iterdir()
    if path.is_dir() and not path.name.startswith(".")
) if SKILLS.is_dir() else []
if skill_dirs != ["fbt-quickstart"]:
    fail(f"skills pack must contain only fbt-quickstart for now; found {skill_dirs}")

if (SKILLS / "package.json").exists():
    package = json.loads(read_text(SKILLS / "package.json"))
    if package.get("name") != "fbt-agent-skills":
        fail("skills/package.json name must be fbt-agent-skills")
    if package.get("license") != "Apache-2.0":
        fail("skills/package.json license must be Apache-2.0")
    keywords = set(package.get("keywords", []))
    for keyword in ("agent-skills", "claude-skills", "tanstack-intent"):
        if keyword not in keywords:
            fail(f"skills/package.json keywords must include {keyword}")
    files = set(package.get("files", []))
    for entry in ("*/SKILL.md", "README.md", "LICENSE", ".claude-plugin/"):
        if entry not in files:
            fail(f"skills/package.json files must include {entry}")

if (SKILLS / ".claude-plugin" / "marketplace.json").exists():
    marketplace = json.loads(read_text(SKILLS / ".claude-plugin" / "marketplace.json"))
    paths = [item.get("path") for item in marketplace.get("skills", [])]
    if paths != ["fbt-quickstart"]:
        fail("marketplace skills must list only fbt-quickstart")

if (SKILLS / "README.md").exists():
    readme = read_text(SKILLS / "README.md")
    for needle in (
        "npx skills add nyuta01/fbt",
        "npx skills add nyuta01/fbt --skill fbt-quickstart",
        "npm install --save-dev fbt-agent-skills",
        "fbt-quickstart",
    ):
        if needle not in readme:
            fail(f"skills/README.md must include {needle!r}")

skill_path = SKILLS / "fbt-quickstart" / "SKILL.md"
if skill_path.exists():
    text = read_text(skill_path)
    match = re.match(r"\A---\n(.*?)\n---\n", text, flags=re.S)
    if not match:
        fail("fbt-quickstart/SKILL.md must start with YAML frontmatter")
    else:
        frontmatter = match.group(1)
        if "name: fbt-quickstart" not in frontmatter:
            fail("fbt-quickstart frontmatter name must be fbt-quickstart")
        if "description:" not in frontmatter:
            fail("fbt-quickstart frontmatter must include description")
    for section in (
        "## When this skill applies",
        "## Procedure",
        "## Verify",
    ):
        if section not in text:
            fail(f"fbt-quickstart must include {section}")
    for command in (
        "fbt init knowledge_ops --template support",
        "fbt doctor --project-dir knowledge_ops",
        "fbt plan --project-dir knowledge_ops --select tag:support",
        "fbt build --project-dir knowledge_ops --select tag:support",
        "fbt artifact explain case_summaries --project-dir knowledge_ops",
    ):
        if command not in text:
            fail(f"fbt-quickstart must include command {command!r}")

if errors:
    print("agent-skills-check: errors found", file=sys.stderr)
    for error in errors:
        print(f"  {error}", file=sys.stderr)
    sys.exit(1)

print("agent-skills-check: ok")
