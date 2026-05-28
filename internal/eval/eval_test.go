package eval

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nyuta01/fbt/internal/manifest"
)

func TestRunForCandidatePassesRequiredSections(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "out/index.md", "# Summary\n\nBody\n")
	transform := manifest.TransformResource{
		UniqueID: "transform.knowledge_ops.case_summaries",
		Evals:    []string{"eval.knowledge_ops.required_sections"},
	}
	evals := map[string]manifest.EvalResource{
		"eval.knowledge_ops.required_sections": {
			UniqueID:         "eval.knowledge_ops.required_sections",
			EvalType:         "deterministic",
			Config:           map[string]any{"sections": []any{"Summary"}},
			GrantsConfidence: "structural",
		},
	}

	outcome, err := RunForCandidate(root, transform, evals, "artifact_version.knowledge_ops.case_summaries.sha256_abc", "transform_run.run_1", "out")
	if err != nil {
		t.Fatalf("run eval: %v", err)
	}
	if len(outcome.Results) != 1 || outcome.Results[0].Status != "pass" {
		t.Fatalf("expected pass result, got %+v", outcome.Results)
	}
	if outcome.Confidence != "structural" {
		t.Fatalf("unexpected confidence grant: %s", outcome.Confidence)
	}
}

func TestRunForCandidateFailsMissingSection(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "out/index.md", "# Other\n")
	transform := manifest.TransformResource{
		UniqueID: "transform.knowledge_ops.case_summaries",
		Evals:    []string{"eval.knowledge_ops.required_sections"},
	}
	evals := map[string]manifest.EvalResource{
		"eval.knowledge_ops.required_sections": {
			UniqueID: "eval.knowledge_ops.required_sections",
			EvalType: "deterministic",
			Config:   map[string]any{"sections": []any{"Summary"}},
		},
	}

	outcome, err := RunForCandidate(root, transform, evals, "artifact_version.knowledge_ops.case_summaries.sha256_abc", "transform_run.run_1", "out")
	if err == nil {
		t.Fatal("expected eval failure")
	}
	if len(outcome.Results) != 1 || outcome.Results[0].Status != "fail" {
		t.Fatalf("expected fail result, got %+v", outcome.Results)
	}
}

func TestRunForCandidateSkipsDelegatedEval(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "out.md", "# Summary\n")
	transform := manifest.TransformResource{
		UniqueID: "transform.knowledge_ops.case_summaries",
		Evals:    []string{"eval.knowledge_ops.semantic_check"},
	}
	evals := map[string]manifest.EvalResource{
		"eval.knowledge_ops.semantic_check": {
			UniqueID: "eval.knowledge_ops.semantic_check",
			EvalType: "semantic",
			Runner:   "runner.knowledge_ops.openai.responses",
		},
	}

	outcome, err := RunForCandidate(root, transform, evals, "artifact_version.knowledge_ops.case_summaries.sha256_abc", "transform_run.run_1", "out.md")
	if err != nil {
		t.Fatalf("run eval: %v", err)
	}
	if len(outcome.Results) != 1 || outcome.Results[0].Status != "skipped" {
		t.Fatalf("expected skipped result, got %+v", outcome.Results)
	}
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
