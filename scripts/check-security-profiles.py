#!/usr/bin/env python3
from __future__ import annotations

from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


def require(path: Path, needle: str) -> None:
    text = path.read_text(encoding="utf-8")
    if needle not in text:
        raise SystemExit(f"{path.relative_to(ROOT)} missing required security profile text: {needle}")


def main() -> int:
    profiles = ROOT / "docs" / "security" / "os-sandbox-profiles.md"
    spec = ROOT / "docs" / "security-and-conformance-spec.md"
    adapters = ROOT / "docs" / "runner-adapters.md"
    docs_page = ROOT / "apps" / "docs" / "src" / "content" / "docs" / "reference" / "security.mdx"
    astro = ROOT / "apps" / "docs" / "astro.config.mjs"

    for path in [profiles, spec, adapters, docs_page]:
        if not path.exists():
            raise SystemExit(f"required security profile file missing: {path.relative_to(ROOT)}")

    for needle in [
        "fbt core does not implement an OS sandbox",
        "CI Sandbox",
        "Container With Read-Only Sources",
        "--network=none",
        "Linux Namespace And Seccomp Wrapper",
        "macOS Local Isolation",
        "Daily Ops Security Handoff",
        "FBT_SECURITY_PROFILE",
        "FBT_SECURITY_TEST_SECRET",
        "Provider SDKs and agent runtimes stay outside core",
        "target/",
        ".fbt/",
    ]:
        require(profiles, needle)

    for needle in [
        "docs/security/os-sandbox-profiles.md",
        "OS-level sandboxing is an external execution profile",
        "tests/conformance/run.py",
    ]:
        require(spec, needle)

    for needle in [
        "OS Sandbox Profiles",
        "fbt core does not implement an OS sandbox",
        "adapter fail-closed mapping + OS sandbox profile",
    ]:
        require(adapters, needle)

    require(docs_page, "docs/security/os-sandbox-profiles.md")
    require(astro, 'slug: "reference/security"')

    print("security-profiles-check: ok")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
