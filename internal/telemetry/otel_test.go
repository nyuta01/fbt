package telemetry

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/nyuta01/fbt/internal/artifact"
	"github.com/nyuta01/fbt/internal/manifest"
	"github.com/nyuta01/fbt/internal/state"
)

func TestOTLPTraces(t *testing.T) {
	version := state.ArtifactVersion{
		VersionID:  "artifact_version.knowledge_ops.case_summaries.sha256_abc",
		ArtifactID: "artifact.knowledge_ops.case_summaries",
		Descriptor: artifact.Descriptor{
			Digest:       "sha256:abc",
			ArtifactType: "fbt.artifact.markdown_directory.v1",
		},
	}
	payload := OTLPTraces(OTLPInput{
		Manifest: manifest.Manifest{
			Metadata: manifest.Metadata{ProjectName: "knowledge_ops"},
			Transforms: map[string]manifest.TransformResource{
				"transform.knowledge_ops.case_summaries": {
					UniqueID: "transform.knowledge_ops.case_summaries",
					Name:     "case_summaries",
					Runner:   "runner.knowledge_ops.openai.responses",
					Policy:   "policy.knowledge_ops.support_agent_scope",
					Model: map[string]any{
						"provider": "local",
						"name":     "mock-gpt",
					},
				},
			},
		},
		ArtifactVersions: state.ArtifactVersionsIndex{
			ArtifactVersions: map[string]state.ArtifactVersion{version.VersionID: version},
		},
		FBTVersion: "test",
		RunResults: []map[string]any{
			{
				"record_type":   "invocation_started",
				"invocation_id": "inv_1",
				"started_at":    "2026-05-28T00:00:00Z",
				"command":       "build",
				"project_name":  "knowledge_ops",
			},
			{
				"record_type":        "transform_run",
				"invocation_id":      "inv_1",
				"run_id":             "transform_run.run_1",
				"transform_id":       "transform.knowledge_ops.case_summaries",
				"status":             "success",
				"started_at":         "2026-05-28T00:00:01Z",
				"completed_at":       "2026-05-28T00:00:02Z",
				"committed_versions": []any{version.VersionID},
				"usage": map[string]any{
					"gen_ai.usage.input_tokens":  100,
					"gen_ai.usage.output_tokens": 20,
					"fbt.usage.total_tokens":     120,
				},
				"events": []any{
					map[string]any{
						"event_type": "usage",
						"time":       "2026-05-28T00:00:02Z",
						"attributes": map[string]any{
							"gen_ai.usage.input_tokens": 100,
							"fbt.usage.total_tokens":    120,
						},
					},
				},
			},
			{
				"record_type":   "invocation_completed",
				"invocation_id": "inv_1",
				"completed_at":  "2026-05-28T00:00:03Z",
				"status":        "success",
			},
		},
	})

	if len(payload.ResourceSpans) != 1 || len(payload.ResourceSpans[0].ScopeSpans) != 1 {
		t.Fatalf("unexpected OTLP shape: %+v", payload)
	}
	spans := payload.ResourceSpans[0].ScopeSpans[0].Spans
	if len(spans) != 2 {
		t.Fatalf("expected invocation and transform spans, got %+v", spans)
	}
	if spans[1].ParentSpanID != spans[0].SpanID {
		t.Fatalf("transform span should be child of invocation span: %+v", spans)
	}
	if len(spans[1].TraceID) != 32 || len(spans[1].SpanID) != 16 {
		t.Fatalf("unexpected trace/span IDs: %+v", spans[1])
	}
	if !hasAttribute(spans[1].Attributes, "gen_ai.request.model", "mock-gpt") {
		t.Fatalf("missing model attribute: %+v", spans[1].Attributes)
	}
	if !hasAttribute(spans[1].Attributes, "fbt.artifact.ids", "artifact.knowledge_ops.case_summaries") {
		t.Fatalf("missing artifact attribute: %+v", spans[1].Attributes)
	}
	if len(spans[1].Events) != 1 || spans[1].Events[0].Name != "usage" {
		t.Fatalf("missing usage event: %+v", spans[1].Events)
	}

	var out bytes.Buffer
	if err := WriteOTLPJSON(&out, payload); err != nil {
		t.Fatal(err)
	}
	var decoded TracesData
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("expected valid OTLP JSON: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), `"resourceSpans"`) {
		t.Fatalf("expected resourceSpans JSON: %s", out.String())
	}
}

func TestOTLPTracesIncludesFailedTransformErrors(t *testing.T) {
	payload := OTLPTraces(OTLPInput{
		Manifest: manifest.Manifest{
			Metadata: manifest.Metadata{ProjectName: "knowledge_ops"},
			Transforms: map[string]manifest.TransformResource{
				"transform.knowledge_ops.case_summaries": {
					UniqueID: "transform.knowledge_ops.case_summaries",
					Name:     "case_summaries",
					Runner:   "runner.knowledge_ops.openai.responses",
				},
			},
		},
		FBTVersion: "test",
		RunResults: []map[string]any{
			{
				"record_type":   "invocation_started",
				"invocation_id": "inv_failed",
				"started_at":    "2026-05-28T00:00:00Z",
				"command":       "build",
			},
			{
				"record_type":   "transform_run",
				"invocation_id": "inv_failed",
				"run_id":        "transform_run.run_failed",
				"transform_id":  "transform.knowledge_ops.case_summaries",
				"status":        "eval_failed",
				"started_at":    "2026-05-28T00:00:01Z",
				"completed_at":  "2026-05-28T00:00:02Z",
				"error": map[string]any{
					"kind":    "eval_failed",
					"message": "eval failed: required sections",
				},
			},
			{
				"record_type":   "invocation_completed",
				"invocation_id": "inv_failed",
				"completed_at":  "2026-05-28T00:00:03Z",
				"status":        "failed",
			},
		},
	})
	spans := payload.ResourceSpans[0].ScopeSpans[0].Spans
	if len(spans) != 2 {
		t.Fatalf("expected invocation and transform spans, got %+v", spans)
	}
	if spans[0].Status.Code != 2 || spans[1].Status.Code != 2 {
		t.Fatalf("expected failed statuses, got %+v %+v", spans[0].Status, spans[1].Status)
	}
	if !hasAttribute(spans[1].Attributes, "error.type", "eval_failed") {
		t.Fatalf("missing error attribute: %+v", spans[1].Attributes)
	}
	if len(spans[1].Events) != 1 || spans[1].Events[0].Name != "exception" {
		t.Fatalf("missing exception event: %+v", spans[1].Events)
	}
}

func hasAttribute(attributes []KeyValue, key, contains string) bool {
	for _, attribute := range attributes {
		if attribute.Key != key {
			continue
		}
		data, _ := json.Marshal(attribute.Value)
		return strings.Contains(string(data), contains)
	}
	return false
}
