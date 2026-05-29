#!/usr/bin/env python3
"""Validate the daily_qa_ops source-window readiness manifest."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

VALID_MODES = {"new_items_only", "cumulative", "correction", "deletion", "backfill"}


def fail(message: str) -> None:
    print(f"source-window: error: {message}", file=sys.stderr)
    raise SystemExit(4)


def load_manifest(path: Path) -> dict:
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except FileNotFoundError:
        fail(f"readiness manifest not found: {path}")
    except json.JSONDecodeError as exc:
        fail(f"readiness manifest is not JSON: {exc}")
    if not isinstance(data, dict):
        fail("readiness manifest must be a JSON object")
    return data


def regular_files(path: Path) -> list[Path]:
    if not path.exists():
        fail(f"source path does not exist: {path}")
    if not path.is_dir():
        fail(f"source path is not a directory: {path}")
    return sorted(p for p in path.rglob("*") if p.is_file() and not p.name.startswith("."))


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--project-dir", required=True)
    parser.add_argument("--ready-file", required=True)
    args = parser.parse_args()

    project_dir = Path(args.project_dir).resolve()
    ready_file = Path(args.ready_file)
    if not ready_file.is_absolute():
        ready_file = (project_dir / ready_file).resolve()

    manifest = load_manifest(ready_file)
    if manifest.get("schema_version") != 1:
        fail("schema_version must be 1")
    window_id = manifest.get("window_id")
    if not isinstance(window_id, str) or not window_id:
        fail("window_id is required")
    mode = manifest.get("mode")
    if mode not in VALID_MODES:
        fail(f"mode must be one of {', '.join(sorted(VALID_MODES))}")
    if not isinstance(manifest.get("prepared_at"), str) or not manifest["prepared_at"]:
        fail("prepared_at is required")
    sources = manifest.get("sources")
    if not isinstance(sources, list) or not sources:
        fail("sources must be a non-empty array")

    print("Source Window")
    print(f"  window_id  {window_id}")
    print(f"  mode       {mode}")
    print(f"  prepared   {manifest['prepared_at']}")

    for source in sources:
        if not isinstance(source, dict):
            fail("each source must be an object")
        name = source.get("name")
        rel_path = source.get("path")
        min_files = source.get("min_files", 1)
        if not isinstance(name, str) or not name:
            fail("source.name is required")
        if not isinstance(rel_path, str) or not rel_path:
            fail(f"{name}: source.path is required")
        if not isinstance(min_files, int) or min_files < 0:
            fail(f"{name}: min_files must be a non-negative integer")
        source_path = (project_dir / rel_path).resolve()
        try:
            source_path.relative_to(project_dir)
        except ValueError:
            fail(f"{name}: source path escapes project directory")
        files = regular_files(source_path)
        if len(files) < min_files:
            fail(f"{name}: expected at least {min_files} files, found {len(files)}")
        print(f"  ok source  {name} files={len(files)} path={rel_path}")

    print("  ok policy  ingestion owns windowing; fbt owns build receipts")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
