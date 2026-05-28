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
