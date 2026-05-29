package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nyuta01/fbt/internal/artifact"
	"github.com/nyuta01/fbt/internal/config"
	"github.com/nyuta01/fbt/internal/manifest"
)

func TestWriteSnapshotFilesAtomically(t *testing.T) {
	store := Open(filepath.Join(t.TempDir(), ".fbt", "state"))
	snapshot := Snapshot{
		Metadata: Metadata{FBTSchemaVersion: StateSchemaVersion, ProjectName: "knowledge_ops"},
		CurrentArtifacts: map[string]ArtifactPointer{
			"artifact.knowledge_ops.report": {
				ArtifactID:       "artifact.knowledge_ops.report",
				CurrentVersionID: "artifact_version.knowledge_ops.report.sha256_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				CurrentDigest:    "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				LogicalPath:      "target/artifacts/report.md",
			},
		},
		LatestRuns: map[string]LatestRun{},
	}
	if err := store.WriteState(snapshot); err != nil {
		t.Fatalf("write state: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(store.Dir, "state.json"))
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	var decoded Snapshot
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("state JSON is invalid: %v", err)
	}
	if decoded.Metadata.ProjectName != "knowledge_ops" {
		t.Fatalf("unexpected project name: %q", decoded.Metadata.ProjectName)
	}
}

func TestWriteManifest(t *testing.T) {
	store := Open(filepath.Join(t.TempDir(), ".fbt", "state"))
	m := manifest.Manifest{
		Metadata:         manifest.Metadata{FBTSchemaVersion: manifest.SchemaVersion, ProjectName: "knowledge_ops"},
		Sources:          map[string]manifest.SourceResource{},
		Artifacts:        map[string]manifest.ArtifactResource{},
		ArtifactVersions: map[string]any{},
		Transforms:       map[string]manifest.TransformResource{},
		TransformAssets:  map[string]manifest.TransformAssetResource{},
		Policies:         map[string]manifest.PolicyResource{},
		Evals:            map[string]manifest.EvalResource{},
		Runners:          map[string]manifest.RunnerResource{},
		ParentMap:        map[string][]string{},
		ChildMap:         map[string][]string{},
		Selectors:        map[string]config.SelectorDefinition{},
		Disabled:         map[string]any{},
		StateSnapshot:    map[string]any{},
		Files:            map[string]manifest.FileResource{},
	}
	if err := store.WriteManifest(m); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store.Dir, "manifest.json")); err != nil {
		t.Fatalf("manifest not written: %v", err)
	}
}

func TestAppendRunResultsJSONL(t *testing.T) {
	store := Open(filepath.Join(t.TempDir(), ".fbt", "state"))
	if err := store.AppendRunResult(map[string]any{"record_type": "invocation_started", "invocation_id": "inv_1"}); err != nil {
		t.Fatalf("append first record: %v", err)
	}
	if err := store.AppendRunResult(map[string]any{"record_type": "invocation_completed", "invocation_id": "inv_1"}); err != nil {
		t.Fatalf("append second record: %v", err)
	}
	records, err := store.ReadRunResults()
	if err != nil {
		t.Fatalf("read run results: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
}

func TestLockPreventsConcurrentAcquireAndReleases(t *testing.T) {
	store := Open(filepath.Join(t.TempDir(), ".fbt", "state"))
	lock, err := store.AcquireLock("inv_1", time.Hour)
	if err != nil {
		t.Fatalf("acquire lock: %v", err)
	}
	if _, err := store.AcquireLock("inv_2", time.Hour); err == nil {
		t.Fatal("expected second lock acquire to fail")
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("release lock: %v", err)
	}
	if _, err := store.AcquireLock("inv_2", time.Hour); err != nil {
		t.Fatalf("expected acquire after release: %v", err)
	}
}

func TestAcquireLockReplacesStaleLock(t *testing.T) {
	store := Open(filepath.Join(t.TempDir(), ".fbt", "state"))
	if err := store.Ensure(); err != nil {
		t.Fatal(err)
	}
	stale := LockInfo{
		InvocationID: "old",
		AcquiredAt:   time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339),
		PID:          1,
	}
	data, err := json.Marshal(stale)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store.Dir, ".lock"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AcquireLock("new", time.Hour); err != nil {
		t.Fatalf("expected stale lock replacement: %v", err)
	}
}

func TestPutArtifactVersionIsIdempotentAndImmutable(t *testing.T) {
	store := Open(filepath.Join(t.TempDir(), ".fbt", "state"))
	size := int64(5)
	version := ArtifactVersion{
		VersionID:   "artifact_version.knowledge_ops.report.sha256_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		ArtifactID:  "artifact.knowledge_ops.report",
		LogicalPath: "target/artifacts/report.md",
		StoragePath: "target/artifacts/report.md",
		Descriptor: artifact.Descriptor{
			MediaType:    "text/markdown; charset=utf-8",
			Digest:       "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			Size:         &size,
			ArtifactType: "fbt.artifact.markdown_document.v1",
		},
	}
	if err := store.PutArtifactVersion(version); err != nil {
		t.Fatalf("put version: %v", err)
	}
	if err := store.PutArtifactVersion(version); err != nil {
		t.Fatalf("idempotent put should pass: %v", err)
	}
	version.StoragePath = "target/artifacts/other.md"
	if err := store.PutArtifactVersion(version); err == nil {
		t.Fatal("expected changed artifact version to be rejected")
	}
}

func TestPutEvaluationResult(t *testing.T) {
	store := Open(filepath.Join(t.TempDir(), ".fbt", "state"))
	artifactVersionID := "artifact_version.knowledge_ops.report.sha256_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	result := EvaluationResult{
		ResultID:          "evaluation_result.knowledge_ops.required_sections.1",
		EvalID:            "eval.knowledge_ops.required_sections",
		ArtifactVersionID: artifactVersionID,
		TransformRunID:    "transform_run.run_1",
		Status:            "pass",
	}
	if err := store.PutEvaluationResult(result); err != nil {
		t.Fatalf("put eval result: %v", err)
	}
	results, err := store.ReadEvaluationResults()
	if err != nil {
		t.Fatalf("read eval results: %v", err)
	}
	if results.EvaluationResults[result.ResultID].Status != "pass" {
		t.Fatalf("eval result not stored: %+v", results.EvaluationResults)
	}
}

func TestPutPolicyDecision(t *testing.T) {
	store := Open(filepath.Join(t.TempDir(), ".fbt", "state"))
	decision := PolicyDecision{
		DecisionID:        "policy_decision.knowledge_ops.report.1",
		PolicyID:          "policy.knowledge_ops.scope",
		TransformID:       "transform.knowledge_ops.report",
		TransformRunID:    "transform_run.run_1",
		ArtifactVersionID: "artifact_version.knowledge_ops.report.sha256_abc",
		Status:            "denied",
		Checks: []PolicyCheck{
			{Name: "write_scope", Status: "fail", Message: "outside write scope"},
		},
		DecidedAt: "2026-05-28T00:00:00Z",
	}
	if err := store.PutPolicyDecision(decision); err != nil {
		t.Fatalf("put policy decision: %v", err)
	}
	decisions, err := store.ReadPolicyDecisions()
	if err != nil {
		t.Fatalf("read policy decisions: %v", err)
	}
	got := decisions.PolicyDecisions[decision.DecisionID]
	if got.Status != "denied" || len(got.Checks) != 1 || got.Checks[0].Message == "" {
		t.Fatalf("policy decision not stored: %+v", decisions.PolicyDecisions)
	}
}

func TestBuildRetentionReportSummarizesLocalState(t *testing.T) {
	root := t.TempDir()
	store := Open(filepath.Join(root, ".fbt", "state"))
	currentVersionID := "artifact_version.knowledge_ops.report.sha256_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	historicalVersionID := "artifact_version.knowledge_ops.report.sha256_abcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd"
	writeFile(t, root, ".fbt/artifacts/current/content/index.md", "# Current\n")
	if err := store.PutArtifactVersion(ArtifactVersion{
		VersionID:   currentVersionID,
		ArtifactID:  "artifact.knowledge_ops.report",
		LogicalPath: "target/artifacts/report/",
		StoragePath: ".fbt/artifacts/current/content",
		Descriptor: artifact.Descriptor{
			Digest:       "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			ArtifactType: "fbt.artifact.markdown_directory.v1",
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.PutArtifactVersion(ArtifactVersion{
		VersionID:   historicalVersionID,
		ArtifactID:  "artifact.knowledge_ops.report",
		LogicalPath: "target/artifacts/report/",
		StoragePath: ".fbt/artifacts/missing/content",
		Descriptor: artifact.Descriptor{
			Digest:       "sha256:abcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd",
			ArtifactType: "fbt.artifact.markdown_directory.v1",
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.WriteState(Snapshot{
		CurrentArtifacts: map[string]ArtifactPointer{
			"artifact.knowledge_ops.report": {
				ArtifactID:       "artifact.knowledge_ops.report",
				CurrentVersionID: currentVersionID,
			},
		},
		LatestRuns: map[string]LatestRun{},
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendRunResult(map[string]any{"record_type": "invocation_started"}); err != nil {
		t.Fatal(err)
	}

	report, err := BuildRetentionReport(root, store)
	if err != nil {
		t.Fatalf("retention report: %v", err)
	}
	if report.Policy != "keep_all" {
		t.Fatalf("unexpected policy: %s", report.Policy)
	}
	if report.ArtifactVersions != 2 || report.CurrentVersions != 1 || report.HistoricalVersions != 1 {
		t.Fatalf("unexpected version counts: %+v", report)
	}
	if report.ArchiveUnit != "state_and_artifacts" || len(report.ArchiveRoots) != 2 {
		t.Fatalf("unexpected archive unit: %+v", report)
	}
	if report.PruneSupported || !report.DryRunRequired {
		t.Fatalf("unexpected prune safety flags: %+v", report)
	}
	if len(report.ProtectedVersionIDs) != 1 || report.ProtectedVersionIDs[0] != currentVersionID {
		t.Fatalf("unexpected protected versions: %+v", report.ProtectedVersionIDs)
	}
	if report.RunRecords != 1 {
		t.Fatalf("unexpected run record count: %+v", report)
	}
	if len(report.MissingStorage) != 1 || report.MissingStorage[0] != historicalVersionID {
		t.Fatalf("unexpected missing storage: %+v", report.MissingStorage)
	}
	if report.StateBytes == 0 || report.ArtifactBytes == 0 {
		t.Fatalf("expected byte counts, got %+v", report)
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
