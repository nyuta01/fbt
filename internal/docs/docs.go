package docs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nyuta01/fbt/internal/manifest"
	"github.com/nyuta01/fbt/internal/state"
)

type Options struct {
	OutputDir string
}

type Result struct {
	OutputDir string `json:"output_dir"`
	IndexPath string `json:"index_path"`
}

func Generate(projectDir string, m manifest.Manifest, store state.Store, options Options) (Result, error) {
	outputDir := options.OutputDir
	if outputDir == "" {
		outputDir = filepath.Join(projectDir, "target", "docs")
	}
	if !filepath.IsAbs(outputDir) {
		outputDir = filepath.Join(projectDir, outputDir)
	}
	snapshot, err := store.ReadState()
	if err != nil {
		return Result{}, err
	}
	versions, err := store.ReadArtifactVersions()
	if err != nil {
		return Result{}, err
	}
	evals, err := store.ReadEvaluationResults()
	if err != nil {
		return Result{}, err
	}
	approvals, err := store.ReadApprovals()
	if err != nil {
		return Result{}, err
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return Result{}, err
	}
	indexPath := filepath.Join(outputDir, "index.md")
	content := render(m, snapshot, versions, evals, approvals)
	if err := os.WriteFile(indexPath, []byte(content), 0o644); err != nil {
		return Result{}, err
	}
	return Result{OutputDir: outputDir, IndexPath: indexPath}, nil
}

func render(m manifest.Manifest, snapshot state.Snapshot, versions state.ArtifactVersionsIndex, evals state.EvaluationResultsIndex, approvals state.ApprovalIndex) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", m.Metadata.ProjectName)
	fmt.Fprintf(&b, "- generated_manifest: `%s`\n", m.Metadata.GeneratedAt)
	fmt.Fprintf(&b, "- sources: %d\n", len(m.Sources))
	fmt.Fprintf(&b, "- transforms: %d\n", len(m.Transforms))
	fmt.Fprintf(&b, "- artifacts: %d\n\n", len(m.Artifacts))

	b.WriteString("## Transforms\n\n")
	for _, id := range sortedTransformIDs(m.Transforms) {
		transform := m.Transforms[id]
		fmt.Fprintf(&b, "### `%s`\n\n", id)
		fmt.Fprintf(&b, "- runner: `%s`\n", transform.Runner)
		fmt.Fprintf(&b, "- type: `%s`\n", transform.TransformType)
		if len(transform.Model) > 0 {
			fmt.Fprintf(&b, "- model: `%s`\n", compactJSON(transform.Model))
		}
		if len(transform.Tools) > 0 {
			fmt.Fprintf(&b, "- tools: `%s`\n", strings.Join(transform.Tools, ", "))
		}
		fmt.Fprintf(&b, "- outputs: `%s`\n", strings.Join(outputIDs(transform), "`, `"))
		fmt.Fprintf(&b, "- evals: `%s`\n\n", strings.Join(transform.Evals, "`, `"))
	}

	b.WriteString("## Current Artifacts\n\n")
	for _, id := range sortedPointerIDs(snapshot.CurrentArtifacts) {
		pointer := snapshot.CurrentArtifacts[id]
		fmt.Fprintf(&b, "- `%s`\n", id)
		fmt.Fprintf(&b, "  - version: `%s`\n", pointer.CurrentVersionID)
		fmt.Fprintf(&b, "  - path: `%s`\n", pointer.LogicalPath)
		fmt.Fprintf(&b, "  - confidence: `%s`\n", pointer.Confidence)
		fmt.Fprintf(&b, "  - approval_status: `%s`\n", pointer.ApprovalStatus)
	}
	b.WriteString("\n## Artifact Versions\n\n")
	for _, id := range sortedVersionIDs(versions.ArtifactVersions) {
		version := versions.ArtifactVersions[id]
		fmt.Fprintf(&b, "- `%s`\n", id)
		fmt.Fprintf(&b, "  - artifact: `%s`\n", version.ArtifactID)
		fmt.Fprintf(&b, "  - digest: `%s`\n", version.Descriptor.Digest)
		fmt.Fprintf(&b, "  - generated_by: `%s`\n", version.GeneratedBy)
	}
	b.WriteString("\n## Evaluation Results\n\n")
	for _, id := range sortedEvalResultIDs(evals.EvaluationResults) {
		result := evals.EvaluationResults[id]
		fmt.Fprintf(&b, "- `%s`: `%s` for `%s`\n", result.EvalID, result.Status, result.ArtifactVersionID)
	}
	b.WriteString("\n## Review Approvals\n\n")
	for _, id := range sortedApprovalIDs(approvals.Approvals) {
		approval := approvals.Approvals[id]
		fmt.Fprintf(&b, "- `%s`: `%s`", id, approval.Status)
		if approval.ReviewGroup != "" {
			fmt.Fprintf(&b, " by `%s`", approval.ReviewGroup)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func outputIDs(transform manifest.TransformResource) []string {
	outputs := make([]string, 0, len(transform.Outputs))
	for _, output := range transform.Outputs {
		outputs = append(outputs, output.UniqueID)
	}
	sort.Strings(outputs)
	return outputs
}

func compactJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(data)
}

func sortedTransformIDs(values map[string]manifest.TransformResource) []string {
	ids := make([]string, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func sortedPointerIDs(values map[string]state.ArtifactPointer) []string {
	ids := make([]string, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func sortedVersionIDs(values map[string]state.ArtifactVersion) []string {
	ids := make([]string, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func sortedEvalResultIDs(values map[string]state.EvaluationResult) []string {
	ids := make([]string, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func sortedApprovalIDs(values map[string]state.Approval) []string {
	ids := make([]string, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
