package runner

import (
	"testing"

	"github.com/nyuta01/fbt/internal/protocol"
)

func TestValidateCapabilities(t *testing.T) {
	result := protocol.InitializeResult{
		Protocol: map[string]any{"version": "0.1"},
		Capabilities: map[string]any{
			"transform_types":   []any{"llm"},
			"artifact_types":    []any{"markdown_directory"},
			"output_candidates": true,
		},
	}
	diagnostics := ValidateCapabilities(result, []CapabilityRequirement{{
		TransformID:             "transform.project.case_summaries",
		TransformType:           "llm",
		ArtifactTypes:           []string{"markdown_directory"},
		RequireOutputCandidates: true,
	}})
	if HasErrors(diagnostics) || !hasDiagnosticCode(diagnostics, "RUNNER_CAPABILITIES_OK") {
		t.Fatalf("expected compatible diagnostics, got %+v", diagnostics)
	}
}

func TestValidateCapabilitiesRejectsMissingArtifactType(t *testing.T) {
	result := protocol.InitializeResult{
		Protocol: map[string]any{"version": "0.1"},
		Capabilities: map[string]any{
			"transform_types":   []any{"llm"},
			"artifact_types":    []any{"text"},
			"output_candidates": true,
		},
	}
	diagnostics := ValidateCapabilities(result, []CapabilityRequirement{{
		TransformID:             "transform.project.case_summaries",
		TransformType:           "llm",
		ArtifactTypes:           []string{"markdown_directory"},
		RequireOutputCandidates: true,
	}})
	if !HasErrors(diagnostics) || !hasDiagnosticCode(diagnostics, "RUNNER_CAPABILITY_ARTIFACT_TYPE_MISSING") {
		t.Fatalf("expected artifact capability error, got %+v", diagnostics)
	}
}
