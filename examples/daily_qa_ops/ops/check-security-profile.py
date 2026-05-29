#!/usr/bin/env python3
"""Validate the daily ops security profile and scan handoff files for secrets."""

from __future__ import annotations

import argparse
import os
import sys
from pathlib import Path

VALID_PROFILES = {"local-trusted", "ci-sandbox", "container-readonly", "network-denied"}
SECRET_ENV_NAMES = ["OPENAI_API_KEY", "ANTHROPIC_API_KEY", "GEMINI_API_KEY", "FBT_SECURITY_TEST_SECRET"]


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--project-dir", required=True)
    parser.add_argument("--run-id", required=True)
    args = parser.parse_args()

    project_dir = Path(args.project_dir).resolve()
    run_id = args.run_id
    profile = os.getenv("FBT_SECURITY_PROFILE", "ci-sandbox")
    if profile not in VALID_PROFILES:
        print(f"security-profile: error: unsupported profile {profile}", file=sys.stderr)
        return 6

    scan_roots = [
        project_dir / "target/ops/runs" / run_id,
        project_dir / "target/ops/publish" / run_id,
    ]
    secret_values = []
    for name in SECRET_ENV_NAMES:
        value = os.getenv(name, "")
        if len(value) >= 8:
            secret_values.append((name, value))

    leaks: list[str] = []
    for root in scan_roots:
        if not root.exists():
            continue
        for path in root.rglob("*"):
            if not path.is_file():
                continue
            try:
                text = path.read_text(encoding="utf-8")
            except UnicodeDecodeError:
                continue
            for name, value in secret_values:
                if value in text:
                    leaks.append(f"{name} leaked in {path.relative_to(project_dir)}")

    print("Security Profile")
    print(f"  profile   {profile}")
    print("  sandbox   external execution profile; fbt core does not manage OS sandboxing")
    print("  secrets   scanned run and publish handoff files for configured secret values")
    if leaks:
        for leak in leaks:
            print(f"  FAIL      {leak}")
        return 6
    print("  result    pass")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
