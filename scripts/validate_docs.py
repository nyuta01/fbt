#!/usr/bin/env python3
"""Validate fbt documentation shape and local Markdown links."""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent

errors: list[str] = []


def fail(message: str) -> None:
    errors.append(message)


def exists(relative: str) -> bool:
    return (ROOT / relative).exists()


def read_text(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")


REQUIRED_DOCS = [
    "README.md",
    "docs/design-doc.md",
    "docs/spec.md",
    "docs/project-config-spec.md",
    "docs/cli-reference.md",
    "docs/manifest-spec.md",
    "docs/state-and-run-results-spec.md",
    "docs/runner-protocol-spec.md",
    "docs/schema-and-versioning-spec.md",
    "docs/runner-discovery-spec.md",
    "docs/runner-lockfile-spec.md",
    "docs/security-and-conformance-spec.md",
    "docs/usage-guide.md",
    "docs/examples/knowledge-loop-example.md",
    "docs/research/dbt-core-overview-report.md",
    "docs/research/related-landscape-report.md",
    "docs/research/fbt-naming-and-spec-standards-research.md",
]

for relative in REQUIRED_DOCS:
    if not exists(relative):
        fail(f"missing required docs file: {relative}")


if exists("README.md"):
    readme = read_text("README.md")
    if len(readme.splitlines()) > 220:
        fail("README.md must stay a compact routing document (<= 220 lines)")
    for link_text in (
        "docs/design-doc.md",
        "docs/spec.md",
        "docs/usage-guide.md",
        "docs/runner-protocol-spec.md",
        "docs/schema-and-versioning-spec.md",
        "docs/runner-discovery-spec.md",
        "docs/security-and-conformance-spec.md",
    ):
        if link_text not in readme:
            fail(f"README.md must link {link_text}")


if exists("docs/spec.md"):
    spec = read_text("docs/spec.md")
    for required_text in (
        "artifact_version",
        "transform_run",
        "transform_asset",
        "JSON-RPC 2.0 compatible runner protocol over stdio",
        "Post-MVP Follow-Up Boundaries",
    ):
        if required_text not in spec:
            fail(f"docs/spec.md must include {required_text!r}")


if exists("docs/runner-protocol-spec.md"):
    protocol = read_text("docs/runner-protocol-spec.md")
    for required_text in (
        "JSON-RPC 2.0 compatible messages over stdio",
        "fbt/runTransform",
        "fbt/outputCandidate",
        "$/cancelRequest",
    ):
        if required_text not in protocol:
            fail(f"docs/runner-protocol-spec.md must include {required_text!r}")


def validate_local_markdown_links() -> None:
    for path in [ROOT / "README.md", *sorted((ROOT / "docs").rglob("*.md"))]:
        if not path.exists():
            continue
        text = path.read_text(encoding="utf-8")
        for match in re.finditer(r"\[[^\]]+\]\(([^)]+)\)", text):
            raw_target = match.group(1).strip()
            if (
                raw_target == ""
                or raw_target.startswith("#")
                or raw_target.startswith("http://")
                or raw_target.startswith("https://")
                or raw_target.startswith("mailto:")
                or "://" in raw_target
            ):
                continue

            target_without_fragment = raw_target.split("#", 1)[0]
            if target_without_fragment == "":
                continue

            resolved = (path.parent / target_without_fragment).resolve()
            if resolved.exists() and resolved.is_dir():
                resolved = resolved / "README.md"

            if not resolved.exists():
                fail(f"{path.relative_to(ROOT)}: broken local link {raw_target}")


def validate_english_docs() -> None:
    japanese = re.compile(r"[\u3040-\u30ff\u3400-\u9fff]")
    for path in [ROOT / "README.md", *sorted((ROOT / "docs").rglob("*.md"))]:
        if path.exists() and japanese.search(path.read_text(encoding="utf-8")):
            fail(f"{path.relative_to(ROOT)}: documentation must stay English")

    for path in sorted((ROOT / "docs").rglob("*-ja.md")):
        fail(f"{path.relative_to(ROOT)}: Japanese suffixed docs were retired")


validate_local_markdown_links()
validate_english_docs()

if errors:
    print("validate-docs: errors found", file=sys.stderr)
    for error in errors:
        print(f"  {error}", file=sys.stderr)
    sys.exit(1)

print("validate-docs: ok")
