#!/usr/bin/env python3
"""Validate the authoritative CI builder reference workflow and docs."""

from __future__ import annotations

import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent

REQUIRED = {
    "examples/daily_qa_ops/ops/github-actions-daily-fbt.yml": [
        "FBT_VERSION: v0.2.1",
        "--version \"$FBT_VERSION\"",
        "concurrency:",
        "FBT_SECURITY_PROFILE: ci-sandbox",
        "Upload fbt run evidence",
        "retention-days:",
        "target/ops/archives",
        "target/ops/publish",
        "authoritative builder",
    ],
    "docs/examples/daily-source-operations.md": [
        "Authoritative CI Builder",
        "developers use local",
        "CI is the authoritative builder",
        "pin the fbt version",
        "target/ops/archives",
    ],
    "docs/release.md": [
        "Authoritative CI Builder",
        "install.sh | sh -s -- --version",
        "adapter versions",
        "run bundle",
    ],
}


def main() -> int:
    errors: list[str] = []
    for relative, needles in REQUIRED.items():
        text = (ROOT / relative).read_text(encoding="utf-8")
        for needle in needles:
            if needle not in text:
                errors.append(f"{relative}: missing {needle!r}")
    if errors:
        print("ci-authority-check: errors found", file=sys.stderr)
        for error in errors:
            print(f"  {error}", file=sys.stderr)
        return 1
    print("ci-authority-check: ok")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
