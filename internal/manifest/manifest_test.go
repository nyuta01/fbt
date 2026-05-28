package manifest

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/nyuta01/fbt/internal/parser"
)

func TestBuildCreatesResourceIDsAndGraphMaps(t *testing.T) {
	parseResult := parseFixture(t)
	m, err := Build(parseResult, BuildOptions{GeneratedAt: fixedTime()})
	if err != nil {
		t.Fatalf("build manifest: %v", err)
	}

	transformID := TransformID("knowledge_ops", "case_summaries")
	sourceID := SourceID("knowledge_ops", "support", "raw_tickets")
	assetID := TransformAssetID("knowledge_ops", "case_summary_prompt")
	policyID := PolicyID("knowledge_ops", "support_agent_scope")
	evalID := EvalID("knowledge_ops", "required_case_sections")
	runnerID := RunnerID("knowledge_ops", "openai.responses")
	artifactID := ArtifactID("knowledge_ops", "case_summaries")

	for _, id := range []string{sourceID, assetID, policyID, evalID, runnerID, artifactID, transformID} {
		if !resourceExists(m, id) {
			t.Fatalf("expected manifest resource %s", id)
		}
	}

	for _, parent := range []string{sourceID, assetID, policyID, evalID, runnerID} {
		if !slices.Contains(m.ParentMap[transformID], parent) {
			t.Fatalf("expected %s to be a parent of %s; got %v", parent, transformID, m.ParentMap[transformID])
		}
	}
	if !slices.Contains(m.ChildMap[transformID], artifactID) {
		t.Fatalf("expected transform to have output child artifact, got %v", m.ChildMap[transformID])
	}
}

func TestManifestJSONIsDeterministic(t *testing.T) {
	parseResult := parseFixture(t)
	options := BuildOptions{GeneratedAt: fixedTime(), InvocationID: "inv_test"}
	first, err := Build(parseResult, options)
	if err != nil {
		t.Fatalf("build first manifest: %v", err)
	}
	second, err := Build(parseResult, options)
	if err != nil {
		t.Fatalf("build second manifest: %v", err)
	}
	firstJSON, err := first.JSON()
	if err != nil {
		t.Fatalf("marshal first manifest: %v", err)
	}
	secondJSON, err := second.JSON()
	if err != nil {
		t.Fatalf("marshal second manifest: %v", err)
	}
	if string(firstJSON) != string(secondJSON) {
		t.Fatalf("manifest JSON is not deterministic")
	}
}

func TestSourceFingerprintChangesWhenGlobFilesChange(t *testing.T) {
	root := writeManifestProject(t)
	firstParse, err := parser.ParseProject(parser.Options{ProjectDir: root})
	if err != nil {
		t.Fatalf("parse first fixture: %v", err)
	}
	first, err := Build(firstParse, BuildOptions{GeneratedAt: fixedTime()})
	if err != nil {
		t.Fatalf("build first manifest: %v", err)
	}

	writeFile(t, root, "data/support/tickets/2026-05-29.jsonl", "{\"id\":\"T-2\"}\n")
	secondParse, err := parser.ParseProject(parser.Options{ProjectDir: root})
	if err != nil {
		t.Fatalf("parse second fixture: %v", err)
	}
	second, err := Build(secondParse, BuildOptions{GeneratedAt: fixedTime()})
	if err != nil {
		t.Fatalf("build second manifest: %v", err)
	}

	sourceID := SourceID("knowledge_ops", "support", "raw_tickets")
	if first.Sources[sourceID].Fingerprint.Value == second.Sources[sourceID].Fingerprint.Value {
		t.Fatalf("expected source fingerprint to change when a glob-matched file is added")
	}
	if len(second.Sources[sourceID].ResolvedPaths) != 2 {
		t.Fatalf("expected new resolved source file, got %v", second.Sources[sourceID].ResolvedPaths)
	}
}

func resourceExists(m Manifest, id string) bool {
	_, ok := m.ResourceSummaries()[id]
	return ok
}

func parseFixture(t *testing.T) parser.Result {
	t.Helper()
	root := writeManifestProject(t)
	result, err := parser.ParseProject(parser.Options{ProjectDir: root})
	if err != nil {
		t.Fatalf("parse fixture: %v\n%+v", err, result.Diagnostics)
	}
	return result
}

func fixedTime() time.Time {
	return time.Date(2026, 5, 28, 10, 0, 0, 0, time.UTC)
}

func writeManifestProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, "fs_project.yml", `name: knowledge_ops
config_version: 1
source_paths: ["sources"]
transform_paths: ["transforms"]
asset_paths: ["assets"]
policy_paths: ["policies"]
eval_paths: ["evals"]
artifact_path: "target/artifacts"
runners:
  - name: openai.responses
    type: llm
    protocol: stdio_jsonrpc
    command: fbt-openai-runner
selectors:
  - name: support_daily
    definition:
      method: tag
      value: support
`)
	writeFile(t, root, "data/support/tickets/2026-05-28.jsonl", "{}\n")
	writeFile(t, root, "prompts/case_summary.md", "# Task\n")
	writeFile(t, root, "sources/support.yml", `sources:
  - name: support
    artifacts:
      - name: raw_tickets
        type: jsonl_directory
        path: data/support/tickets/*.jsonl
        tags: ["support", "raw"]
`)
	writeFile(t, root, "assets/assets.yml", `assets:
  - name: case_summary_prompt
    type: prompt
    path: prompts/case_summary.md
`)
	writeFile(t, root, "policies/support.yml", `policies:
  - name: support_agent_scope
    read: ["data/support/"]
    write: [".fbt/work/", "target/artifacts/support/"]
    network: true
`)
	writeFile(t, root, "evals/support.yml", `evals:
  - name: required_case_sections
    type: deterministic
    config:
      sections: ["Summary"]
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
    tags: ["support", "knowledge"]
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
