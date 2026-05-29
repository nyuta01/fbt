#!/usr/bin/env python3
from __future__ import annotations

import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


def require(path: Path, needle: str) -> None:
    text = path.read_text(encoding="utf-8")
    if needle not in text:
        raise SystemExit(f"{path.relative_to(ROOT)} missing required release-plan text: {needle}")


def main() -> int:
    release = ROOT / "docs" / "release.md"
    adapters = ROOT / "docs" / "runner-adapters.md"
    workflow = ROOT / ".github" / "workflows" / "release-adapters.yml"

    for path in [release, adapters, workflow]:
        if not path.exists():
            raise SystemExit(f"required adapter release plan file missing: {path.relative_to(ROOT)}")

    required_tags = [
        "sdk/go/v0.1.0",
        "adapters/command/v0.1.0",
        "adapters/openai/v0.1.0",
        "adapters/codex-cli/v0.1.0",
        "adapters/claude-code/v0.1.0",
    ]
    for tag in required_tags:
        require(release, tag)
        require(adapters, tag)

    for needle in [
        "git tag -s",
        "git tag -v",
        "go mod download -json",
        "GOSUMDB",
        "SHA256SUMS",
        "cosign sign-blob",
        "make adapter-release-plan-check",
    ]:
        require(release, needle)

    for needle in [
        "module-scoped tags",
        "signed annotated tag",
        "SHA256SUMS",
        "runner protocol version",
        "make adapter-install-smoke",
    ]:
        require(adapters, needle)

    for needle in [
        "adapters/**/v*",
        "sdk/go/v*",
        "make adapter-release-plan-check",
        "make official-adapter-smoke",
        "make adapter-install-smoke",
    ]:
        require(workflow, needle)

    print("adapter-release-plan-check: ok")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

