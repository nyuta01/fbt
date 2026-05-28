package policy

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/nyuta01/fbt/internal/artifact"
	"github.com/nyuta01/fbt/internal/manifest"
	"github.com/nyuta01/fbt/internal/security"
)

type Decision struct {
	Status string  `json:"status"`
	Checks []Check `json:"checks"`
}

type Check struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func CheckLogicalArtifactPath(projectDir, artifactPath, outputPath string) error {
	artifactRoot, err := security.ResolveProjectRelative(projectDir, artifactPath)
	if err != nil {
		return err
	}
	outputAbs, err := security.ResolveProjectRelative(projectDir, outputPath)
	if err != nil {
		return err
	}
	return security.RequireWithin(artifactRoot, outputAbs)
}

func CheckWriteScope(projectDir string, scopes []string, outputPath string) error {
	if len(scopes) == 0 {
		return nil
	}
	outputAbs, err := security.ResolveProjectRelative(projectDir, outputPath)
	if err != nil {
		return err
	}
	for _, scope := range scopes {
		scopeAbs, err := security.ResolveProjectRelative(projectDir, scope)
		if err != nil {
			return err
		}
		if security.IsWithin(scopeAbs, outputAbs) {
			return nil
		}
	}
	return fmt.Errorf("output path %s is outside declared write scope", outputPath)
}

func CheckOutputSize(descriptor artifact.Descriptor, maxBytes int64) error {
	if maxBytes <= 0 {
		return nil
	}
	if descriptor.Size != nil && *descriptor.Size > maxBytes {
		return fmt.Errorf("output size %d exceeds limit %d", *descriptor.Size, maxBytes)
	}
	return nil
}

func Timeout(policy manifest.PolicyResource) time.Duration {
	seconds := numberFromMap(policy.Limits, "timeout_seconds")
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func MaxOutputBytes(policy manifest.PolicyResource) int64 {
	return numberFromMap(policy.Limits, "max_output_bytes")
}

func RedactSecrets(text string, secrets map[string]string) string {
	redacted := text
	for name, value := range secrets {
		if value == "" {
			continue
		}
		redacted = strings.ReplaceAll(redacted, value, fmt.Sprintf("${%s}", name))
	}
	return redacted
}

func EvaluateCommit(projectDir, artifactPath string, policyResource *manifest.PolicyResource, output manifest.TransformOutput, descriptor artifact.Descriptor) Decision {
	decision := Decision{Status: "allowed"}
	add := func(name string, err error) {
		if err != nil {
			decision.Status = "denied"
			decision.Checks = append(decision.Checks, Check{Name: name, Status: "fail", Message: err.Error()})
			return
		}
		decision.Checks = append(decision.Checks, Check{Name: name, Status: "pass"})
	}
	add("artifact_path", CheckLogicalArtifactPath(projectDir, artifactPath, output.DeclaredPath))
	if policyResource != nil {
		add("write_scope", CheckWriteScope(projectDir, policyResource.WriteScope, output.DeclaredPath))
		add("max_output_bytes", CheckOutputSize(descriptor, MaxOutputBytes(*policyResource)))
	}
	return decision
}

func numberFromMap(values map[string]any, key string) int64 {
	if values == nil {
		return 0
	}
	switch value := values[key].(type) {
	case int:
		return int64(value)
	case int64:
		return value
	case float64:
		return int64(value)
	case uint64:
		return int64(value)
	}
	return 0
}

func ScopeContains(projectDir string, scope string, path string) bool {
	scopeAbs, err := security.ResolveProjectRelative(projectDir, filepath.Clean(scope))
	if err != nil {
		return false
	}
	pathAbs, err := security.ResolveProjectRelative(projectDir, filepath.Clean(path))
	if err != nil {
		return false
	}
	return security.IsWithin(scopeAbs, pathAbs)
}
