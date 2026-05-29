#!/usr/bin/env python3
"""Check that core release version references stay in sync."""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent


def read_text(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")


def fail(message: str) -> None:
    errors.append(message)


def normalize_version(raw: str) -> str:
    version = raw[1:] if raw.startswith("v") else raw
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        raise ValueError(f"release version must look like X.Y.Z or vX.Y.Z: {raw}")
    return version


errors: list[str] = []

makefile = read_text("Makefile")
match = re.search(r"^VERSION \?= (\d+\.\d+\.\d+)$", makefile, re.MULTILINE)
if not match:
    fail("Makefile must define VERSION ?= X.Y.Z")
    current_version = "0.0.0"
else:
    current_version = match.group(1)

try:
    expected_version = normalize_version(sys.argv[1]) if len(sys.argv) > 1 else current_version
except ValueError as exc:
    print(exc, file=sys.stderr)
    sys.exit(2)

expected_tag = f"v{expected_version}"

if current_version != expected_version:
    fail(f"Makefile VERSION is {current_version}, expected {expected_version}")


def require_text(relative: str, text: str) -> None:
    if text not in read_text(relative):
        fail(f"{relative} must include {text!r}")


def require_json_version(relative: str, keys: list[str]) -> None:
    data = json.loads(read_text(relative))
    value = data
    for key in keys:
        value = value[key]
    if value != expected_version:
        fail(f"{relative}:{'.'.join(keys)} is {value!r}, expected {expected_version!r}")


required_texts = {
    "internal/version/version.go": [f'Version   = "{expected_version}"'],
    "internal/cli/cli_test.go": [
        f'fbt {expected_version}',
        f'"version": "{expected_version}"',
    ],
    "scripts/dist-check.sh": [f'VERSION="${{VERSION:-{expected_version}}}"'],
    "scripts/smoke-cli.sh": [f"^fbt {expected_version}$"],
    "README.md": [f"releases/tag/{expected_tag}"],
    "docs/release.md": [
        expected_tag,
        f"fbt_{expected_version}_darwin_arm64.tar.gz",
        "scripts/release-preflight.sh",
        ".github/workflows/release-core.yml",
    ],
    "docs/cli-reference.md": [f"fbt {expected_version}"],
    "docs/schema-and-versioning-spec.md": [f"`{expected_version}`"],
    "docs/manifest-spec.md": [f'"fbt_version": "{expected_version}"'],
    "docs/standard-export-spec.md": [f'"fbt_version": "{expected_version}"'],
    "apps/docs/src/content/docs/get-started/installation.mdx": [
        f"releases/tag/{expected_tag}"
    ],
    "apps/docs/src/content/docs/get-started/what-you-can-do.mdx": [
        f"fbt {expected_tag}"
    ],
    "apps/docs/src/content/docs/reference/release.mdx": [
        expected_tag,
        "scripts/release-preflight.sh",
    ],
}

for relative, needles in required_texts.items():
    for needle in needles:
        require_text(relative, needle)

require_json_version("apps/docs/package.json", ["version"])
require_json_version("apps/docs/package-lock.json", ["version"])
require_json_version("apps/docs/package-lock.json", ["packages", "", "version"])

release_workflow = read_text(".github/workflows/release-core.yml")
for needle in (
    "scripts/release-preflight.sh --allow-existing-tag",
    "gh release create",
    "--verify-tag",
    "--generate-notes",
    "--fail-on-no-commits",
    "contents: write",
):
    if needle not in release_workflow:
        fail(f".github/workflows/release-core.yml must include {needle!r}")
for action, ref in re.findall(r"uses:\s*(actions/[^@\s]+)@([0-9A-Za-z_.-]+)", release_workflow):
    if not re.fullmatch(r"[0-9a-f]{40}", ref):
        fail(f".github/workflows/release-core.yml must pin {action} by full commit SHA")

verify_workflow = read_text(".github/workflows/verify.yml")
if 'tags:' in verify_workflow:
    fail(".github/workflows/verify.yml should not duplicate core release tag verification")

if errors:
    print("release-version-check: errors found", file=sys.stderr)
    for error in errors:
        print(f"  {error}", file=sys.stderr)
    sys.exit(1)

print(f"release-version-check: ok ({expected_tag})")
