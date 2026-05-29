#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parents[1]
SCHEMA_DIR = ROOT / "schemas"
PROJECT_SCHEMA_PATH = SCHEMA_DIR / "project-config-v1.schema.json"
RESOURCE_SCHEMA_PATH = SCHEMA_DIR / "resource-file-v1.schema.json"


def extract_go_map_keys(path: Path, var_name: str) -> list[str]:
    text = path.read_text(encoding="utf-8")
    match = re.search(rf"var {re.escape(var_name)} = map\[string\][^{{]+{{(?P<body>.*?)\n}}", text, re.S)
    if not match:
        raise SystemExit(f"could not find Go map {var_name} in {path}")
    return sorted(re.findall(r'"([^"]+)"\s*:', match.group("body")))


def extract_string_set_function(path: Path, func_name: str) -> set[str]:
    text = path.read_text(encoding="utf-8")
    match = re.search(rf"func {re.escape(func_name)}\(\) map\[string\]struct{{}} {{(?P<body>.*?)\n}}", text, re.S)
    if not match:
        raise SystemExit(f"could not find Go function {func_name} in {path}")
    return set(re.findall(r'"([^"]+)"', match.group("body")))


def schema_object(
    properties: dict[str, Any],
    *,
    required: list[str] | None = None,
    description: str | None = None,
    additional: bool | dict[str, Any] = False,
) -> dict[str, Any]:
    out: dict[str, Any] = {
        "type": "object",
        "additionalProperties": additional,
        "properties": properties,
    }
    if required:
        out["required"] = required
    if description:
        out["description"] = description
    return out


def array_of(item: dict[str, Any]) -> dict[str, Any]:
    return {"type": "array", "items": item}


STRING = {"type": "string"}
STRING_ARRAY = array_of(STRING)
FREE_OBJECT = {"type": "object", "additionalProperties": True}


def named_resource(properties: dict[str, Any], required: list[str] | None = None) -> dict[str, Any]:
    merged = {"name": STRING, **properties}
    return schema_object(merged, required=required or ["name"])


def build_project_schema() -> dict[str, Any]:
    parser_go = ROOT / "internal" / "parser" / "parser.go"
    config_go = ROOT / "internal" / "config" / "config.go"
    artifact_aliases = extract_go_map_keys(config_go, "artifactTypes")
    transform_types = extract_go_map_keys(parser_go, "transformTypes")

    artifact_type = {
        "description": "Built-in artifact type alias or extension type starting with x.",
        "anyOf": [
            {"enum": artifact_aliases},
            {"type": "string", "pattern": r"^x\..+"},
        ],
    }

    runner = named_resource(
        {
            "type": STRING,
            "protocol": {"type": "string", "enum": ["stdio_jsonrpc"]},
            "command": STRING,
            "args": STRING_ARRAY,
            "cwd": STRING,
            "env": STRING_ARRAY,
            "config": FREE_OBJECT,
            "capabilities": FREE_OBJECT,
        },
        required=["name", "type", "protocol", "command"],
    )

    selector_definition: dict[str, Any] = {
        "type": "object",
        "additionalProperties": False,
        "properties": {
            "method": {"type": "string", "enum": ["tag", "name"]},
            "value": STRING,
            "union": {"type": "array", "items": {"$ref": "#/$defs/selector_definition"}},
        },
    }

    project_schema = {
        "$schema": "https://json-schema.org/draft/2020-12/schema",
        "$id": "https://schemas.fbt.dev/fbt/project-config/v1.schema.json",
        "title": "fbt project config v1",
        "description": "Schema for fs_project.yml. Resource YAML schemas live in resource-file-v1.schema.json.",
        "type": "object",
        "additionalProperties": False,
        "required": ["name"],
        "anyOf": [
            {"required": ["config_version"]},
            {"required": ["config-version"]},
        ],
        "properties": {
            "name": STRING,
            "config_version": {"type": "integer", "const": 1},
            "config-version": {"type": "integer", "const": 1, "deprecated": True},
            "version": STRING,
            "source_paths": STRING_ARRAY,
            "source-paths": {**STRING_ARRAY, "deprecated": True},
            "transform_paths": STRING_ARRAY,
            "transform-paths": {**STRING_ARRAY, "deprecated": True},
            "asset_paths": STRING_ARRAY,
            "asset-paths": {**STRING_ARRAY, "deprecated": True},
            "policy_paths": STRING_ARRAY,
            "policy-paths": {**STRING_ARRAY, "deprecated": True},
            "eval_paths": STRING_ARRAY,
            "eval-paths": {**STRING_ARRAY, "deprecated": True},
            "target_path": STRING,
            "target-path": {**STRING, "deprecated": True},
            "artifact_path": STRING,
            "artifact-path": {**STRING, "deprecated": True},
            "state": schema_object(
                {
                    "backend": {"type": "string", "const": "local"},
                    "path": STRING,
                }
            ),
            "execution": schema_object(
                {
                    "mode": {"type": "string", "const": "local"},
                }
            ),
            "runners": array_of({"$ref": "#/$defs/runner"}),
            "selectors": array_of(
                named_resource(
                    {
                        "definition": {"$ref": "#/$defs/selector_definition"},
                    },
                    required=["name", "definition"],
                )
            ),
            "vars": FREE_OBJECT,
        },
        "$defs": {
            "runner": runner,
            "selector_definition": selector_definition,
            "artifact_type": artifact_type,
            "transform_type": {"enum": transform_types},
        },
    }
    return project_schema


def build_resource_schema() -> dict[str, Any]:
    parser_go = ROOT / "internal" / "parser" / "parser.go"
    config_go = ROOT / "internal" / "config" / "config.go"
    artifact_aliases = extract_go_map_keys(config_go, "artifactTypes")
    transform_types = extract_go_map_keys(parser_go, "transformTypes")
    eval_types = extract_go_map_keys(parser_go, "evalTypes")

    artifact_type = {
        "description": "Built-in artifact type alias or extension type starting with x.",
        "anyOf": [
            {"enum": artifact_aliases},
            {"type": "string", "pattern": r"^x\..+"},
        ],
    }

    runner = named_resource(
        {
            "type": STRING,
            "protocol": {"type": "string", "enum": ["stdio_jsonrpc"]},
            "command": STRING,
            "args": STRING_ARRAY,
            "cwd": STRING,
            "env": STRING_ARRAY,
            "config": FREE_OBJECT,
            "capabilities": FREE_OBJECT,
        },
        required=["name", "type", "protocol", "command"],
    )
    source_artifact = named_resource(
        {
            "type": artifact_type,
            "path": STRING,
            "description": STRING,
            "tags": STRING_ARRAY,
            "tests": array_of({}),
            "meta": FREE_OBJECT,
        },
        required=["name", "type", "path"],
    )
    source = named_resource(
        {
            "description": STRING,
            "artifacts": array_of(source_artifact),
        },
        required=["name", "artifacts"],
    )
    artifact = named_resource(
        {
            "type": artifact_type,
            "path": STRING,
            "contract": FREE_OBJECT,
            "owner": STRING,
            "tags": STRING_ARRAY,
            "meta": FREE_OBJECT,
        },
        required=["name", "type", "path"],
    )
    asset = named_resource(
        {
            "type": STRING,
            "path": STRING,
            "variables": STRING_ARRAY,
            "meta": FREE_OBJECT,
        },
        required=["name", "type", "path"],
    )
    transform_input = schema_object(
        {
            "source": STRING,
            "ref": STRING,
            "require": schema_object(
                {
                    "confidence": STRING,
                    "evals": FREE_OBJECT,
                }
            ),
        }
    )
    transform_output = named_resource(
        {
            "type": artifact_type,
            "path": STRING,
            "contract": FREE_OBJECT,
        },
        required=["name", "type", "path"],
    )
    asset_ref = schema_object(
        {
            "ref": STRING,
            "type": STRING,
            "path": STRING,
        }
    )
    transform = named_resource(
        {
            "type": {"enum": transform_types},
            "runner": STRING,
            "command": STRING_ARRAY,
            "model": FREE_OBJECT,
            "agent": STRING,
            "inputs": array_of(transform_input),
            "outputs": array_of(transform_output),
            "assets": array_of(asset_ref),
            "tools": STRING_ARRAY,
            "policy": STRING,
            "evals": STRING_ARRAY,
            "contract": FREE_OBJECT,
            "tags": STRING_ARRAY,
            "meta": FREE_OBJECT,
        },
        required=["name", "type", "runner", "outputs"],
    )
    policy = named_resource(
        {
            "read": STRING_ARRAY,
            "write": STRING_ARRAY,
            "network": {"type": "boolean"},
            "tools": FREE_OBJECT,
            "limits": FREE_OBJECT,
        }
    )
    eval_schema = named_resource(
        {
            "type": {"enum": eval_types},
            "runner": STRING,
            "config": FREE_OBJECT,
            "grants_confidence": STRING,
        },
        required=["name", "type"],
    )

    return {
        "$schema": "https://json-schema.org/draft/2020-12/schema",
        "$id": "https://schemas.fbt.dev/fbt/resource-file/v1.schema.json",
        "title": "fbt resource file v1",
        "description": "Schema for YAML files under source, transform, asset, policy, and eval paths.",
        "type": "object",
        "additionalProperties": False,
        "minProperties": 1,
        "properties": {
            "sources": array_of(source),
            "artifacts": array_of(artifact),
            "assets": array_of(asset),
            "transforms": array_of(transform),
            "policies": array_of(policy),
            "evals": array_of(eval_schema),
            "runners": array_of(runner),
        },
        "$defs": {
            "artifact_type": artifact_type,
            "runner": runner,
        },
    }


def validate_against_parser_contract(project_schema: dict[str, Any], resource_schema: dict[str, Any]) -> None:
    parser_go = ROOT / "internal" / "parser" / "parser.go"
    project_allowed = extract_string_set_function(parser_go, "allowedProjectFields")
    resource_top_allowed = extract_string_set_function(parser_go, "allowedResourceTopLevelFields")
    runner_allowed = extract_string_set_function(parser_go, "allowedRunnerFields")
    transform_allowed = extract_string_set_function(parser_go, "allowedTransformFields")

    project_props = set(project_schema["properties"])
    if not project_props.issubset(project_allowed):
        raise SystemExit(f"project schema has fields parser does not allow: {sorted(project_props - project_allowed)}")
    if "defaults" in project_props:
        raise SystemExit("project schema must not allow reserved defaults")
    execution_props = set(project_schema["properties"]["execution"]["properties"])
    if execution_props != {"mode"}:
        raise SystemExit(f"execution schema must expose only mode: {sorted(execution_props)}")

    resource_props = set(resource_schema["properties"])
    if resource_props != resource_top_allowed:
        raise SystemExit(f"resource top-level schema drift: {sorted(resource_props ^ resource_top_allowed)}")

    runner_props = set(project_schema["$defs"]["runner"]["properties"])
    if runner_props != runner_allowed:
        raise SystemExit(f"runner schema drift: {sorted(runner_props ^ runner_allowed)}")

    transform_props = set(resource_schema["properties"]["transforms"]["items"]["properties"])
    transform_reserved = {"cache", "review"}
    if transform_props != transform_allowed - transform_reserved:
        raise SystemExit(f"transform schema drift: {sorted(transform_props ^ (transform_allowed - transform_reserved))}")


def render(schema: dict[str, Any]) -> str:
    return json.dumps(schema, indent=2, sort_keys=True) + "\n"


def main() -> int:
    parser = argparse.ArgumentParser(description="Generate or check fbt project config JSON Schemas.")
    parser.add_argument("--write", action="store_true", help="write generated schemas")
    parser.add_argument("--check", action="store_true", help="fail if checked-in schemas are stale")
    args = parser.parse_args()
    if not args.write and not args.check:
        args.check = True

    project_schema = build_project_schema()
    resource_schema = build_resource_schema()
    validate_against_parser_contract(project_schema, resource_schema)
    outputs = {
        PROJECT_SCHEMA_PATH: render(project_schema),
        RESOURCE_SCHEMA_PATH: render(resource_schema),
    }

    if args.write:
        SCHEMA_DIR.mkdir(parents=True, exist_ok=True)
        for path, content in outputs.items():
            path.write_text(content, encoding="utf-8")
    if args.check:
        missing_or_stale: list[str] = []
        for path, content in outputs.items():
            if not path.exists() or path.read_text(encoding="utf-8") != content:
                missing_or_stale.append(str(path.relative_to(ROOT)))
        if missing_or_stale:
            print("stale generated schema files:", ", ".join(missing_or_stale), file=sys.stderr)
            print("run: python3 scripts/generate-project-config-schema.py --write", file=sys.stderr)
            return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

