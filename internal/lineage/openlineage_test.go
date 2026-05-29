package lineage

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/nyuta01/fbt/internal/artifact"
	"github.com/nyuta01/fbt/internal/manifest"
	"github.com/nyuta01/fbt/internal/state"
)

func TestOpenLineageEvents(t *testing.T) {
	version := state.ArtifactVersion{
		VersionID:   "artifact_version.knowledge_ops.case_summaries.sha256_abc",
		ArtifactID:  "artifact.knowledge_ops.case_summaries",
		LogicalPath: "target/artifacts/support/case_summaries/",
		StoragePath: ".fbt/artifacts/sha256/abc",
		GeneratedBy: "transform_run.run_1",
		Confidence:  "structural",
		CommittedAt: "2026-05-28T00:00:00Z",
		Descriptor: artifact.Descriptor{
			MediaType:    "inode/directory",
			Digest:       "sha256:abc",
			ArtifactType: "fbt.artifact.markdown_directory.v1",
			FileCount:    1,
		},
	}
	events := OpenLineageEvents(OpenLineageInput{
		Manifest: manifest.Manifest{
			Metadata: manifest.Metadata{ProjectName: "knowledge_ops", GeneratedAt: "2026-05-28T00:00:00Z"},
			Sources: map[string]manifest.SourceResource{
				"source.knowledge_ops.raw_tickets": {
					UniqueID:     "source.knowledge_ops.raw_tickets",
					Name:         "raw_tickets",
					ArtifactType: "jsonl_directory",
					Path:         "data/support/tickets/*.jsonl",
				},
			},
			Transforms: map[string]manifest.TransformResource{
				"transform.knowledge_ops.case_summaries": {
					UniqueID:      "transform.knowledge_ops.case_summaries",
					Name:          "case_summaries",
					TransformType: "llm",
					Runner:        "runner.knowledge_ops.openai.responses",
					Policy:        "policy.knowledge_ops.support_agent_scope",
					Evals:         []string{"eval.knowledge_ops.required_case_sections"},
					Model:         map[string]any{"provider": "openai", "name": "gpt-test"},
					Inputs: []manifest.TransformInput{
						{Kind: "source", UniqueID: "source.knowledge_ops.raw_tickets", Name: "raw_tickets"},
					},
					Outputs: []manifest.TransformOutput{
						{UniqueID: "artifact.knowledge_ops.case_summaries", Name: "case_summaries"},
					},
				},
			},
		},
		ArtifactVersions: state.ArtifactVersionsIndex{
			ArtifactVersions: map[string]state.ArtifactVersion{version.VersionID: version},
		},
		EvaluationResults: state.EvaluationResultsIndex{
			EvaluationResults: map[string]state.EvaluationResult{
				"eval_result.1": {
					ResultID:          "eval_result.1",
					EvalID:            "eval.knowledge_ops.required_case_sections",
					ArtifactVersionID: version.VersionID,
					Status:            "pass",
				},
			},
		},
	})

	if len(events) != 1 {
		t.Fatalf("expected one event, got %d", len(events))
	}
	event := events[0]
	if event.EventType != "COMPLETE" || event.EventTime != "2026-05-28T00:00:00Z" {
		t.Fatalf("unexpected event basics: %+v", event)
	}
	if event.Job.Namespace != "fbt:knowledge_ops" || event.Job.Name != "transform.knowledge_ops.case_summaries" {
		t.Fatalf("unexpected job: %+v", event.Job)
	}
	if !strings.Contains(event.Run.RunID, "-5") {
		t.Fatalf("expected deterministic v5-like UUID, got %q", event.Run.RunID)
	}
	if len(event.Inputs) != 1 || event.Inputs[0].Name != "source.knowledge_ops.raw_tickets" {
		t.Fatalf("unexpected inputs: %+v", event.Inputs)
	}
	if len(event.Outputs) != 1 || event.Outputs[0].Name != version.ArtifactID {
		t.Fatalf("unexpected outputs: %+v", event.Outputs)
	}
	if _, ok := event.Outputs[0].Facets["fbt_artifact"]; !ok {
		t.Fatalf("missing fbt_artifact facet: %+v", event.Outputs[0].Facets)
	}
	if _, ok := event.Outputs[0].Facets["fbt_evaluations"]; !ok {
		t.Fatalf("missing fbt_evaluations facet: %+v", event.Outputs[0].Facets)
	}

	var out bytes.Buffer
	if err := WriteOpenLineageNDJSON(&out, events); err != nil {
		t.Fatal(err)
	}
	var decoded RunEvent
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &decoded); err != nil {
		t.Fatalf("expected valid NDJSON event: %v\n%s", err, out.String())
	}
	if decoded.SchemaURL != OpenLineageSchemaURL || decoded.Producer != OpenLineageProducer {
		t.Fatalf("unexpected OpenLineage metadata: %+v", decoded)
	}
}

func TestOpenLineageEventsIncludeOrphanedArtifactVersions(t *testing.T) {
	version := state.ArtifactVersion{
		VersionID:   "artifact_version.knowledge_ops.case_summaries.sha256_orphan",
		ArtifactID:  "artifact.knowledge_ops.case_summaries",
		LogicalPath: "target/artifacts/support/case_summaries/",
		StoragePath: ".fbt/artifacts/orphan/content",
		GeneratedBy: "transform_run.run_orphan",
		CommittedAt: "2026-05-28T00:00:00Z",
		Descriptor: artifact.Descriptor{
			Digest:       "sha256:orphan",
			ArtifactType: "fbt.artifact.markdown_directory.v1",
		},
		Materials: []state.Material{
			{ResourceID: "source.knowledge_ops.support.raw_tickets", Digest: "sha256:source"},
		},
	}
	events := OpenLineageEvents(OpenLineageInput{
		Manifest: manifest.Manifest{
			Metadata:   manifest.Metadata{ProjectName: "knowledge_ops", GeneratedAt: "2026-05-28T00:00:00Z"},
			Transforms: map[string]manifest.TransformResource{},
		},
		ArtifactVersions: state.ArtifactVersionsIndex{
			ArtifactVersions: map[string]state.ArtifactVersion{version.VersionID: version},
		},
	})

	if len(events) != 1 {
		t.Fatalf("expected one orphaned event, got %d", len(events))
	}
	event := events[0]
	if event.Job.Name != version.ArtifactID {
		t.Fatalf("expected artifact job name for orphaned event, got %+v", event.Job)
	}
	facet, ok := event.Job.Facets["fbt_job"].(map[string]any)
	if !ok || facet["orphaned"] != true {
		t.Fatalf("expected orphaned job facet, got %+v", event.Job.Facets)
	}
	if len(event.Inputs) != 1 || event.Inputs[0].Name != "source.knowledge_ops.support.raw_tickets" {
		t.Fatalf("expected material input for orphaned event, got %+v", event.Inputs)
	}
}
