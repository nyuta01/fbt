package parser

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestParseProjectValidResources(t *testing.T) {
	root := writeValidProject(t)

	result, err := ParseProject(Options{ProjectDir: root})
	if err != nil {
		t.Fatalf("parse project: %v\n%+v", err, result.Diagnostics)
	}
	if len(result.Sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(result.Sources))
	}
	if len(result.Assets) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(result.Assets))
	}
	if len(result.Transforms) != 1 {
		t.Fatalf("expected 1 transform, got %d", len(result.Transforms))
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %+v", result.Diagnostics)
	}
}

func TestParseProjectKeepsFreeFormContracts(t *testing.T) {
	root := writeValidProject(t)
	writeFile(t, root, "transforms/case.yml", `transforms:
  - name: case_summaries
    type: llm
    runner: openai.responses
    contract:
      runner_prompt: summarize_support_cases_v1
      model_preferences:
        temperature: 0.2
    inputs:
      - source: support.raw_tickets
    outputs:
      - name: case_summaries
        type: markdown_directory
        path: target/artifacts/support/case_summaries/
        contract:
          format: support_case_summary_v1
          required_sections:
            - Summary
            - Next actions
    assets:
      - ref: case_summary_prompt
    policy: support_agent_scope
`)

	result, err := ParseProject(Options{ProjectDir: root})
	if err != nil {
		t.Fatalf("parse project: %v\n%+v", err, result.Diagnostics)
	}
	transform := result.Transforms[0]
	if transform.Contract["runner_prompt"] != "summarize_support_cases_v1" {
		t.Fatalf("expected transform contract to remain free-form, got %+v", transform.Contract)
	}
	outputContract := transform.Outputs[0].Contract
	if outputContract["format"] != "support_case_summary_v1" {
		t.Fatalf("expected output contract to remain free-form, got %+v", outputContract)
	}
	sections := outputContract["required_sections"].([]any)
	if len(sections) != 2 || sections[0] != "Summary" {
		t.Fatalf("expected nested output contract values, got %+v", outputContract)
	}
}

func TestParseProjectRequiresConfigVersion(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "fs_project.yml", "name: demo\n")

	result, err := ParseProject(Options{ProjectDir: root})
	if err == nil {
		t.Fatal("expected parse error")
	}
	assertDiagnostic(t, result.Diagnostics, "CONFIG_VERSION_MISSING")
}

func TestParseProjectRejectsUnsupportedArtifactType(t *testing.T) {
	root := writeValidProject(t)
	writeFile(t, root, "transforms/case.yml", `transforms:
  - name: case_summaries
    type: llm
    runner: openai.responses
    inputs:
      - source: support.raw_tickets
    outputs:
      - name: case_summaries
        type: nope
        path: target/artifacts/support/case_summaries/
    assets:
      - ref: case_summary_prompt
    policy: support_agent_scope
    evals:
      - required_case_sections
`)

	result, err := ParseProject(Options{ProjectDir: root})
	if err == nil {
		t.Fatal("expected parse error")
	}
	assertDiagnostic(t, result.Diagnostics, "ARTIFACT_TYPE_UNSUPPORTED")
}

func TestParseProjectRejectsArtifactPathEscape(t *testing.T) {
	root := writeValidProject(t)
	writeFile(t, root, "transforms/case.yml", `transforms:
  - name: case_summaries
    type: llm
    runner: openai.responses
    inputs:
      - source: support.raw_tickets
    outputs:
      - name: case_summaries
        type: markdown_directory
        path: target/elsewhere/case_summaries/
    assets:
      - ref: case_summary_prompt
    policy: support_agent_scope
    evals:
      - required_case_sections
`)

	result, err := ParseProject(Options{ProjectDir: root})
	if err == nil {
		t.Fatal("expected parse error")
	}
	assertDiagnostic(t, result.Diagnostics, "PATH_OUTSIDE_ARTIFACT_PATH")
}

func TestParseProjectRejectsUnsupportedStateBackend(t *testing.T) {
	root := writeValidProject(t)
	writeFile(t, root, "fs_project.yml", `name: knowledge_ops
config_version: 1
source_paths: ["sources"]
transform_paths: ["transforms"]
asset_paths: ["assets"]
policy_paths: ["policies"]
eval_paths: ["evals"]
artifact_path: "target/artifacts"
state:
  backend: postgres
  path: .fbt/state
runners:
  - name: openai.responses
    type: llm
    protocol: stdio_jsonrpc
    command: fbt-openai-runner
`)

	result, err := ParseProject(Options{ProjectDir: root})
	if err == nil {
		t.Fatal("expected parse error")
	}
	diagnostic := findDiagnostic(t, result.Diagnostics, "STATE_BACKEND_UNSUPPORTED")
	if diagnostic.Hint == "" {
		t.Fatalf("expected state backend hint, got %+v", diagnostic)
	}
}

func TestParseProjectRejectsStatePathEscape(t *testing.T) {
	root := writeValidProject(t)
	writeFile(t, root, "fs_project.yml", `name: knowledge_ops
config_version: 1
source_paths: ["sources"]
transform_paths: ["transforms"]
asset_paths: ["assets"]
policy_paths: ["policies"]
eval_paths: ["evals"]
artifact_path: "target/artifacts"
state:
  backend: local
  path: ../fbt-state
runners:
  - name: openai.responses
    type: llm
    protocol: stdio_jsonrpc
    command: fbt-openai-runner
`)

	result, err := ParseProject(Options{ProjectDir: root})
	if err == nil {
		t.Fatal("expected parse error")
	}
	assertDiagnostic(t, result.Diagnostics, "STATE_PATH_INVALID")
}

func TestParseProjectRejectsUnresolvedRefs(t *testing.T) {
	root := writeValidProject(t)
	writeFile(t, root, "transforms/case.yml", `transforms:
  - name: weekly_support_insights
    type: agent
    runner: langgraph.agent
    inputs:
      - ref: case_summaries
    outputs:
      - name: weekly_support_insights
        type: markdown
        path: target/artifacts/support/weekly_insights.md
    assets:
      - ref: missing_prompt
    policy: support_agent_scope
    evals:
      - missing_eval
`)

	result, err := ParseProject(Options{ProjectDir: root})
	if err == nil {
		t.Fatal("expected parse error")
	}
	assertDiagnostic(t, result.Diagnostics, "REF_UNRESOLVED")
	assertDiagnostic(t, result.Diagnostics, "ASSET_REF_UNRESOLVED")
	assertDiagnostic(t, result.Diagnostics, "EVAL_REF_UNRESOLVED")
	diagnostic := findDiagnostic(t, result.Diagnostics, "REF_UNRESOLVED")
	if diagnostic.Line == 0 {
		t.Fatalf("expected line-oriented diagnostic, got %+v", diagnostic)
	}
	if diagnostic.Hint == "" {
		t.Fatalf("expected actionable hint, got %+v", diagnostic)
	}
}

func TestParseProjectRejectsAgentTransformWithoutPolicy(t *testing.T) {
	root := writeValidProject(t)
	writeFile(t, root, "transforms/case.yml", `transforms:
  - name: weekly_support_insights
    type: agent
    runner: openai.responses
    inputs:
      - source: support.raw_tickets
    outputs:
      - name: weekly_support_insights
        type: markdown
        path: target/artifacts/support/weekly_insights.md
    assets:
      - ref: case_summary_prompt
`)

	result, err := ParseProject(Options{ProjectDir: root})
	if err == nil {
		t.Fatal("expected parse error")
	}
	diagnostic := findDiagnostic(t, result.Diagnostics, "AGENT_POLICY_MISSING")
	if diagnostic.Severity != SeverityError {
		t.Fatalf("expected error severity, got %+v", diagnostic)
	}
}

func TestParseProjectRejectsReviewFields(t *testing.T) {
	root := writeValidProject(t)
	writeFile(t, root, "policies/support.yml", `policies:
  - name: support_agent_scope
    read:
      - data/support/
    write:
      - target/artifacts/support/
    review:
      required: true
`)
	writeFile(t, root, "transforms/case.yml", `transforms:
  - name: case_summaries
    type: llm
    runner: openai.responses
    inputs:
      - source: support.raw_tickets
    outputs:
      - name: case_summaries
        type: markdown_directory
        path: target/artifacts/support/case_summaries/
    review:
      required: true
    policy: support_agent_scope
`)

	result, err := ParseProject(Options{ProjectDir: root})
	if err == nil {
		t.Fatal("expected parse error")
	}
	diagnostic := findDiagnostic(t, result.Diagnostics, "REVIEW_UNSUPPORTED")
	if diagnostic.Line == 0 {
		t.Fatalf("expected line-oriented diagnostic, got %+v", diagnostic)
	}
	if diagnostic.Hint == "" {
		t.Fatalf("expected actionable hint, got %+v", diagnostic)
	}
}

func TestParseProjectRejectsReservedProjectConfigFields(t *testing.T) {
	root := writeValidProject(t)
	writeFile(t, root, "fs_project.yml", `name: knowledge_ops
config_version: 1
source_paths: ["sources"]
transform_paths: ["transforms"]
asset_paths: ["assets"]
policy_paths: ["policies"]
eval_paths: ["evals"]
execution:
  mode: local
  max_workers: 4
  fail_fast: false
defaults:
  cache:
    mode: reuse_if_same_inputs
  confidence:
    minimum: structural
runners:
  - name: openai.responses
    type: llm
    protocol: stdio_jsonrpc
    command: fbt-openai-runner
`)

	result, err := ParseProject(Options{ProjectDir: root})
	if err == nil {
		t.Fatal("expected parse error")
	}
	diagnostic := findDiagnostic(t, result.Diagnostics, "CONFIG_FIELD_RESERVED")
	if diagnostic.Line == 0 || diagnostic.Hint == "" {
		t.Fatalf("expected line-oriented actionable diagnostic, got %+v", diagnostic)
	}
}

func TestParseProjectRejectsReservedTransformCache(t *testing.T) {
	root := writeValidProject(t)
	writeFile(t, root, "transforms/case.yml", `transforms:
  - name: case_summaries
    type: llm
    runner: openai.responses
    cache:
      mode: reuse_if_same_inputs
    inputs:
      - source: support.raw_tickets
    outputs:
      - name: case_summaries
        type: markdown_directory
        path: target/artifacts/support/case_summaries/
    policy: support_agent_scope
`)

	result, err := ParseProject(Options{ProjectDir: root})
	if err == nil {
		t.Fatal("expected parse error")
	}
	diagnostic := findDiagnostic(t, result.Diagnostics, "CONFIG_FIELD_RESERVED")
	if diagnostic.Resource != "case_summaries" || diagnostic.Line == 0 || diagnostic.Hint == "" {
		t.Fatalf("expected transform cache diagnostic, got %+v", diagnostic)
	}
}

func TestParseProjectRejectsUnknownYAMLFields(t *testing.T) {
	cases := []struct {
		name     string
		mutate   func(t *testing.T, root string)
		resource string
	}{
		{
			name: "project top-level typo",
			mutate: func(t *testing.T, root string) {
				writeFile(t, root, "fs_project.yml", `name: knowledge_ops
config_version: 1
sorce_paths: ["sources"]
transform_paths: ["transforms"]
asset_paths: ["assets"]
policy_paths: ["policies"]
eval_paths: ["evals"]
runners:
  - name: openai.responses
    type: llm
    protocol: stdio_jsonrpc
    command: fbt-openai-runner
`)
			},
			resource: "project",
		},
		{
			name: "runner field typo",
			mutate: func(t *testing.T, root string) {
				writeFile(t, root, "fs_project.yml", `name: knowledge_ops
config_version: 1
source_paths: ["sources"]
transform_paths: ["transforms"]
asset_paths: ["assets"]
policy_paths: ["policies"]
eval_paths: ["evals"]
runners:
  - name: openai.responses
    type: llm
    protocol: stdio_jsonrpc
    cmd: fbt-openai-runner
`)
			},
			resource: "openai.responses",
		},
		{
			name: "source artifact field typo",
			mutate: func(t *testing.T, root string) {
				writeFile(t, root, "sources/support.yml", `sources:
  - name: support
    artifacts:
      - name: raw_tickets
        type: jsonl_directory
        pth: data/support/tickets/*.jsonl
`)
			},
			resource: "support.raw_tickets",
		},
		{
			name: "transform field typo",
			mutate: func(t *testing.T, root string) {
				writeFile(t, root, "transforms/case.yml", `transforms:
  - name: case_summaries
    type: llm
    runner: openai.responses
    modle:
      provider: openai
    inputs:
      - source: support.raw_tickets
    outputs:
      - name: case_summaries
        type: markdown_directory
        path: target/artifacts/support/case_summaries/
    policy: support_agent_scope
`)
			},
			resource: "case_summaries",
		},
		{
			name: "policy field typo",
			mutate: func(t *testing.T, root string) {
				writeFile(t, root, "policies/support.yml", `policies:
  - name: support_agent_scope
    read:
      - data/support/
    write:
      - .fbt/work/
      - target/artifacts/support/
    netwrok: true
`)
			},
			resource: "support_agent_scope",
		},
		{
			name: "eval field typo",
			mutate: func(t *testing.T, root string) {
				writeFile(t, root, "evals/support.yml", `evals:
  - name: required_case_sections
    type: deterministic
    grant_confidence: structural
    config:
      sections:
        - Summary
`)
			},
			resource: "required_case_sections",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := writeValidProject(t)
			tc.mutate(t, root)

			result, err := ParseProject(Options{ProjectDir: root})
			if err == nil {
				t.Fatal("expected parse error")
			}
			diagnostic := findDiagnostic(t, result.Diagnostics, "YAML_FIELD_UNKNOWN")
			if diagnostic.Resource != tc.resource || diagnostic.Line == 0 || diagnostic.Hint == "" {
				t.Fatalf("expected unknown field diagnostic for %q, got %+v", tc.resource, diagnostic)
			}
		})
	}
}

func writeValidProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, "fs_project.yml", `name: knowledge_ops
config_version: 1
version: 0.1.0

source_paths: ["sources"]
transform_paths: ["transforms"]
asset_paths: ["assets"]
policy_paths: ["policies"]
eval_paths: ["evals"]

target_path: "target"
artifact_path: "target/artifacts"

runners:
  - name: openai.responses
    type: llm
    protocol: stdio_jsonrpc
    command: fbt-openai-runner
`)
	writeFile(t, root, "data/support/tickets/2026-05-28.jsonl", "{}\n")
	writeFile(t, root, "prompts/case_summary.md", "# Task\n")
	writeFile(t, root, "sources/support.yml", `sources:
  - name: support
    artifacts:
      - name: raw_tickets
        type: jsonl_directory
        path: data/support/tickets/*.jsonl
`)
	writeFile(t, root, "assets/assets.yml", `assets:
  - name: case_summary_prompt
    type: prompt
    path: prompts/case_summary.md
`)
	writeFile(t, root, "policies/support.yml", `policies:
  - name: support_agent_scope
    read:
      - data/support/
    write:
      - .fbt/work/
      - target/artifacts/support/
    network: true
`)
	writeFile(t, root, "evals/support.yml", `evals:
  - name: required_case_sections
    type: deterministic
    config:
      sections:
        - Summary
`)
	writeFile(t, root, "transforms/case.yml", `transforms:
  - name: case_summaries
    type: llm
    runner: openai.responses
    inputs:
      - source: support.raw_tickets
    outputs:
      - name: case_summaries
        type: markdown_directory
        path: target/artifacts/support/case_summaries/
    assets:
      - ref: case_summary_prompt
    policy: support_agent_scope
    evals:
      - required_case_sections
`)
	return root
}

func writeFile(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertDiagnostic(t *testing.T, diagnostics []Diagnostic, code string) {
	t.Helper()
	_ = findDiagnostic(t, diagnostics, code)
}

func findDiagnostic(t *testing.T, diagnostics []Diagnostic, code string) Diagnostic {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return diagnostic
		}
	}
	t.Fatalf("expected diagnostic %s, got %+v", code, diagnostics)
	return Diagnostic{}
}

func TestParseProjectReturnsDiagnosticsError(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "fs_project.yml", "name: demo\n")
	_, err := ParseProject(Options{ProjectDir: root})
	var diagnosticsErr DiagnosticsError
	if !errors.As(err, &diagnosticsErr) {
		t.Fatalf("expected DiagnosticsError, got %T", err)
	}
}
