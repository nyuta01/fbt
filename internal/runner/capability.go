package runner

import (
	"errors"
	"fmt"
	"strings"

	"github.com/nyuta01/fbt/internal/protocol"
)

var ErrCapabilityIncompatible = errors.New("runner capability incompatible")

type CapabilityRequirement struct {
	TransformID             string
	TransformType           string
	ArtifactTypes           []string
	RequireOutputCandidates bool
}

func ValidateCapabilities(result protocol.InitializeResult, requirements []CapabilityRequirement) []Diagnostic {
	var diagnostics []Diagnostic
	version, _ := result.Protocol["version"].(string)
	if version != "0.1" {
		diagnostics = append(diagnostics, Diagnostic{Severity: "error", Code: "RUNNER_PROTOCOL_VERSION_UNSUPPORTED", Message: fmt.Sprintf("runner protocol version %q is not supported", version)})
	}
	if requiresOutputCandidates(requirements) && !boolCapability(result.Capabilities, "output_candidates") {
		diagnostics = append(diagnostics, Diagnostic{Severity: "error", Code: "RUNNER_CAPABILITY_OUTPUT_CANDIDATES_MISSING", Message: "runner must support output_candidates"})
	}

	transformTypes := stringSet(result.Capabilities["transform_types"])
	artifactTypes := stringSet(result.Capabilities["artifact_types"])
	for _, requirement := range requirements {
		if requirement.TransformType != "" && !containsCapability(transformTypes, requirement.TransformType) {
			diagnostics = append(diagnostics, Diagnostic{
				Severity: "error",
				Code:     "RUNNER_CAPABILITY_TRANSFORM_TYPE_MISSING",
				Message:  fmt.Sprintf("runner does not support transform type %q required by %s", requirement.TransformType, requirement.TransformID),
			})
		}
		for _, artifactType := range requirement.ArtifactTypes {
			if artifactType == "" || containsCapability(artifactTypes, artifactType) {
				continue
			}
			diagnostics = append(diagnostics, Diagnostic{
				Severity: "error",
				Code:     "RUNNER_CAPABILITY_ARTIFACT_TYPE_MISSING",
				Message:  fmt.Sprintf("runner does not support artifact type %q required by %s", artifactType, requirement.TransformID),
			})
		}
	}
	if !HasErrors(diagnostics) {
		diagnostics = append(diagnostics, Diagnostic{Severity: "info", Code: "RUNNER_CAPABILITIES_OK", Message: "runner capabilities satisfy configured transforms"})
	}
	return diagnostics
}

func requiresOutputCandidates(requirements []CapabilityRequirement) bool {
	for _, requirement := range requirements {
		if requirement.RequireOutputCandidates {
			return true
		}
	}
	return false
}

func CapabilityError(diagnostics []Diagnostic) error {
	var messages []string
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == "error" {
			messages = append(messages, diagnostic.Message)
		}
	}
	if len(messages) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrCapabilityIncompatible, strings.Join(messages, "; "))
}

func containsCapability(values map[string]struct{}, value string) bool {
	_, ok := values[value]
	return ok
}

func boolCapability(values map[string]any, key string) bool {
	value, ok := values[key]
	if !ok {
		return false
	}
	typed, ok := value.(bool)
	return ok && typed
}

func stringSet(value any) map[string]struct{} {
	out := map[string]struct{}{}
	switch typed := value.(type) {
	case []string:
		for _, item := range typed {
			out[item] = struct{}{}
		}
	case []any:
		for _, item := range typed {
			if text, ok := item.(string); ok {
				out[text] = struct{}{}
			}
		}
	}
	return out
}
