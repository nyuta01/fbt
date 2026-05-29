package policy

import (
	"testing"

	"github.com/nyuta01/fbt/internal/artifact"
	"github.com/nyuta01/fbt/internal/manifest"
)

func TestCheckLogicalArtifactPath(t *testing.T) {
	root := t.TempDir()
	if err := CheckLogicalArtifactPath(root, "target/artifacts", "target/artifacts/report.md"); err != nil {
		t.Fatalf("expected artifact path to pass: %v", err)
	}
	if err := CheckLogicalArtifactPath(root, "target/artifacts", "target/other/report.md"); err == nil {
		t.Fatal("expected path outside artifact root to fail")
	}
}

func TestCheckWriteScope(t *testing.T) {
	root := t.TempDir()
	if err := CheckWriteScope(root, []string{"target/artifacts/support/"}, "target/artifacts/support/report.md"); err != nil {
		t.Fatalf("expected write scope to pass: %v", err)
	}
	if err := CheckWriteScope(root, []string{"target/artifacts/support/"}, "target/artifacts/legal/report.md"); err == nil {
		t.Fatal("expected write scope to fail")
	}
}

func TestEvaluateCommitDeniesSizeLimit(t *testing.T) {
	size := int64(10)
	policyResource := manifest.PolicyResource{Limits: map[string]any{"max_output_bytes": 5}}
	decision := EvaluateCommit(t.TempDir(), "target/artifacts", &policyResource, manifest.TransformOutput{
		Name:         "report",
		ArtifactType: "markdown",
		DeclaredPath: "target/artifacts/report.md",
	}, artifact.Descriptor{Size: &size})
	if decision.Status != "denied" {
		t.Fatalf("expected denied decision, got %+v", decision)
	}
}

func TestEvaluateCommitDeniesDirectorySizeLimit(t *testing.T) {
	size := int64(12)
	policyResource := manifest.PolicyResource{Limits: map[string]any{"max_output_bytes": 5}}
	decision := EvaluateCommit(t.TempDir(), "target/artifacts", &policyResource, manifest.TransformOutput{
		Name:         "report",
		ArtifactType: "markdown_directory",
		DeclaredPath: "target/artifacts/report",
	}, artifact.Descriptor{Size: &size, FileCount: 2})
	if decision.Status != "denied" {
		t.Fatalf("expected directory size denial, got %+v", decision)
	}
}

func TestTimeoutAndRedaction(t *testing.T) {
	policyResource := manifest.PolicyResource{Limits: map[string]any{"timeout_seconds": 12}}
	if got := Timeout(policyResource).Seconds(); got != 12 {
		t.Fatalf("unexpected timeout: %v", got)
	}
	redacted := RedactSecrets("token=abc123", map[string]string{"TOKEN": "abc123"})
	if redacted != "token=${TOKEN}" {
		t.Fatalf("unexpected redaction: %q", redacted)
	}
}
