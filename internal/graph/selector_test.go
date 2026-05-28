package graph

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nyuta01/fbt/internal/config"
	"github.com/nyuta01/fbt/internal/manifest"
	"github.com/nyuta01/fbt/internal/parser"
)

func TestSelectByResourceTypeTagAndPath(t *testing.T) {
	m := selectorManifest(t)

	transforms, err := Select(m, Selector{Method: "resource_type", Value: "transform"})
	if err != nil {
		t.Fatalf("select transform: %v", err)
	}
	assertContains(t, transforms, manifest.TransformID("knowledge_ops", "case_summaries"))

	tagged, err := Select(m, Selector{Method: "tag", Value: "support"})
	if err != nil {
		t.Fatalf("select tag: %v", err)
	}
	assertContains(t, tagged, manifest.TransformID("knowledge_ops", "case_summaries"))
	assertContains(t, tagged, manifest.SourceID("knowledge_ops", "support", "raw_tickets"))

	byPath, err := Select(m, Selector{Method: "path", Value: "target/artifacts/support"})
	if err != nil {
		t.Fatalf("select path: %v", err)
	}
	assertContains(t, byPath, manifest.ArtifactID("knowledge_ops", "case_summaries"))
}

func TestSelectParentChildAndUnionDefinition(t *testing.T) {
	m := selectorManifest(t)
	transformID := manifest.TransformID("knowledge_ops", "case_summaries")
	sourceID := manifest.SourceID("knowledge_ops", "support", "raw_tickets")
	artifactID := manifest.ArtifactID("knowledge_ops", "case_summaries")

	parents, err := Select(m, Selector{Method: "parent", Value: transformID})
	if err != nil {
		t.Fatalf("select parents: %v", err)
	}
	assertContains(t, parents, sourceID)

	children, err := Select(m, Selector{Method: "child", Value: transformID})
	if err != nil {
		t.Fatalf("select children: %v", err)
	}
	assertContains(t, children, artifactID)

	selected, err := SelectDefinition(m, config.SelectorDefinition{
		Union: []config.SelectorDefinition{
			{Method: "resource_type", Value: "transform"},
			{Method: "resource_type", Value: "source"},
		},
	})
	if err != nil {
		t.Fatalf("select union: %v", err)
	}
	assertContains(t, selected, transformID)
	assertContains(t, selected, sourceID)
}

func TestSelectTransformsGraphOperators(t *testing.T) {
	m := selectorManifest(t)
	caseID := manifest.TransformID("knowledge_ops", "case_summaries")
	weeklyID := manifest.TransformID("knowledge_ops", "weekly_support_insights")
	sourceID := manifest.SourceID("knowledge_ops", "support", "raw_tickets")
	artifactID := manifest.ArtifactID("knowledge_ops", "case_summaries")

	downstream, err := SelectTransforms(m, "case_summaries+")
	if err != nil {
		t.Fatalf("select downstream: %v", err)
	}
	assertMapContains(t, downstream, caseID)
	assertMapContains(t, downstream, weeklyID)
	assertMapNotContains(t, downstream, sourceID)
	assertMapNotContains(t, downstream, artifactID)

	upstream, err := SelectTransforms(m, "+weekly_support_insights")
	if err != nil {
		t.Fatalf("select upstream: %v", err)
	}
	assertMapContains(t, upstream, caseID)
	assertMapContains(t, upstream, weeklyID)
	assertMapNotContains(t, upstream, sourceID)
	assertMapNotContains(t, upstream, artifactID)

	both, err := SelectTransforms(m, "+case_summaries+")
	if err != nil {
		t.Fatalf("select both directions: %v", err)
	}
	assertMapContains(t, both, caseID)
	assertMapContains(t, both, weeklyID)
}

func TestSelectTransformsRejectsAmbiguousName(t *testing.T) {
	m := manifest.Manifest{
		Transforms: map[string]manifest.TransformResource{
			"transform.test.first":  {UniqueID: "transform.test.first", ResourceType: "transform", Name: "duplicate"},
			"transform.test.second": {UniqueID: "transform.test.second", ResourceType: "transform", Name: "duplicate"},
		},
	}
	_, err := SelectTransforms(m, "duplicate")
	if err == nil || !strings.Contains(err.Error(), "ambiguous selector") {
		t.Fatalf("expected ambiguous selector error, got %v", err)
	}
}

func selectorManifest(t *testing.T) manifest.Manifest {
	t.Helper()
	root := manifestFixture(t)
	parseResult, err := parser.ParseProject(parser.Options{ProjectDir: root})
	if err != nil {
		t.Fatalf("parse fixture: %v\n%+v", err, parseResult.Diagnostics)
	}
	m, err := manifest.Build(parseResult, manifest.BuildOptions{
		GeneratedAt: time.Date(2026, 5, 28, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("build manifest: %v", err)
	}
	return m
}

func assertContains(t *testing.T, values []string, want string) {
	t.Helper()
	if !slices.Contains(values, want) {
		t.Fatalf("expected %s in %v", want, values)
	}
}

func assertMapContains(t *testing.T, values map[string]struct{}, want string) {
	t.Helper()
	if _, ok := values[want]; !ok {
		t.Fatalf("expected %s in %v", want, values)
	}
}

func assertMapNotContains(t *testing.T, values map[string]struct{}, unwanted string) {
	t.Helper()
	if _, ok := values[unwanted]; ok {
		t.Fatalf("did not expect %s in %v", unwanted, values)
	}
}

func manifestFixture(t *testing.T) string {
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
  - name: weekly_support_insights
    type: llm
    runner: openai.responses
    inputs:
      - ref: case_summaries
        require:
          confidence: structural
    outputs:
      - name: weekly_support_insights
        type: markdown
        path: target/artifacts/support/weekly_insights.md
    policy: support_agent_scope
    tags: ["insights"]
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
