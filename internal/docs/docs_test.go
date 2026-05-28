package docs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nyuta01/fbt/internal/artifact"
	"github.com/nyuta01/fbt/internal/manifest"
	"github.com/nyuta01/fbt/internal/state"
)

func TestGenerateWritesStaticMarkdownDocs(t *testing.T) {
	root := t.TempDir()
	store := state.Open(filepath.Join(root, ".fbt", "state"))
	version := state.ArtifactVersion{
		VersionID:  "artifact_version.knowledge_ops.report.sha256_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		ArtifactID: "artifact.knowledge_ops.report",
		Descriptor: artifact.Descriptor{
			Digest: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		},
		GeneratedBy: "transform_run.run_1",
		SemanticDescriptor: map[string]any{
			"text_normalized_v1": map[string]any{"digest": "sha256:semantic"},
		},
	}
	if err := store.PutArtifactVersion(version); err != nil {
		t.Fatal(err)
	}
	if err := store.WriteState(state.Snapshot{
		CurrentArtifacts: map[string]state.ArtifactPointer{
			version.ArtifactID: {
				ArtifactID:       version.ArtifactID,
				CurrentVersionID: version.VersionID,
				LogicalPath:      "target/artifacts/report.md",
				Confidence:       "structural",
			},
		},
		LatestRuns: map[string]state.LatestRun{},
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.PutEvaluationResult(state.EvaluationResult{
		ResultID:          "evaluation_result.knowledge_ops.required.1",
		EvalID:            "eval.knowledge_ops.required",
		ArtifactVersionID: version.VersionID,
		Status:            "pass",
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.PutPolicyDecision(state.PolicyDecision{
		DecisionID:        "policy_decision.knowledge_ops.report.1",
		PolicyID:          "policy.knowledge_ops.scope",
		TransformID:       "transform.knowledge_ops.report",
		TransformRunID:    version.GeneratedBy,
		ArtifactVersionID: version.VersionID,
		Status:            "allowed",
		Checks: []state.PolicyCheck{
			{Name: "write_scope", Status: "pass"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	m := manifest.Manifest{
		Metadata: manifest.Metadata{ProjectName: "knowledge_ops"},
		Transforms: map[string]manifest.TransformResource{
			"transform.knowledge_ops.report": {
				UniqueID:      "transform.knowledge_ops.report",
				TransformType: "llm",
				Runner:        "runner.knowledge_ops.local.llm",
				Model:         map[string]any{"name": "mock"},
				Outputs: []manifest.TransformOutput{
					{UniqueID: version.ArtifactID},
				},
				Evals: []string{"eval.knowledge_ops.required"},
			},
		},
		Artifacts: map[string]manifest.ArtifactResource{version.ArtifactID: {UniqueID: version.ArtifactID}},
		Sources:   map[string]manifest.SourceResource{},
	}

	result, err := Generate(root, m, store, Options{})
	if err != nil {
		t.Fatalf("generate docs: %v", err)
	}
	data, err := os.ReadFile(result.IndexPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, want := range []string{"# knowledge_ops", "confidence", "structural", "eval.knowledge_ops.required", "Policy Decisions", "write_scope", "text_normalized_v1"} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected %q in docs:\n%s", want, content)
		}
	}
}
