#!/usr/bin/env python3
"""Validate optional runner lockfile design stays validator-only."""

from __future__ import annotations

import json
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent

errors: list[str] = []


def require_text(relative: str, text: str) -> None:
    content = (ROOT / relative).read_text(encoding="utf-8")
    if text not in content:
        errors.append(f"{relative} must include {text!r}")


for required in (
    "docs/runner-lockfile-spec.md",
    "schemas/fbt-lock-v1.schema.json",
):
    if not (ROOT / required).exists():
        errors.append(f"missing {required}")


if (ROOT / "docs/runner-lockfile-spec.md").exists():
    for phrase in (
        "fbt.lock.json",
        "Core must not become a package manager",
        "download runners or adapters",
        "fbt doctor",
        "lockfile changes participate in dirty-state selection",
        "no core command downloads, installs, or resolves adapter packages",
    ):
        require_text("docs/runner-lockfile-spec.md", phrase)

if (ROOT / "schemas/fbt-lock-v1.schema.json").exists():
    schema = json.loads((ROOT / "schemas/fbt-lock-v1.schema.json").read_text(encoding="utf-8"))
    if schema.get("$id") != "https://schemas.fbt.dev/fbt/runner-lock/v1.json":
        errors.append("schemas/fbt-lock-v1.schema.json must use the runner-lock v1 schema id")
    required = set(schema.get("required", []))
    for field in ("fbt_schema_version", "lockfile_version", "runners"):
        if field not in required:
            errors.append(f"lockfile schema must require {field}")

for relative in (
    "README.md",
    "docs/spec.md",
    "docs/project-config-spec.md",
    "docs/runner-discovery-spec.md",
    "docs/runner-adapters.md",
    "docs/schema-and-versioning-spec.md",
):
    require_text(relative, "runner-lockfile-spec.md" if relative != "docs/schema-and-versioning-spec.md" else "runner-lock/v1.json")

if errors:
    print("runner-lockfile-spec-check: errors found", file=sys.stderr)
    for error in errors:
        print(f"  {error}", file=sys.stderr)
    sys.exit(1)

print("runner-lockfile-spec-check: ok")
