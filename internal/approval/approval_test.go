package approval

import (
	"path/filepath"
	"testing"

	"github.com/nyuta01/fbt/internal/artifact"
	"github.com/nyuta01/fbt/internal/state"
)

func TestApprovePromotesCurrentPointer(t *testing.T) {
	store, version := writeApprovalState(t)
	if err := PutPending(store, version, "support_leads"); err != nil {
		t.Fatalf("put pending: %v", err)
	}

	status, err := Approve(store, "case_summaries", "", "reviewer", "looks good")
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if status.Status != "approved" || status.Confidence != "reviewed" {
		t.Fatalf("unexpected status: %+v", status)
	}
	snapshot, err := store.ReadState()
	if err != nil {
		t.Fatal(err)
	}
	pointer := snapshot.CurrentArtifacts[version.ArtifactID]
	if pointer.ApprovalStatus != "approved" || pointer.Confidence != "reviewed" {
		t.Fatalf("pointer was not promoted: %+v", pointer)
	}
}

func TestRejectMarksCurrentPointerRejected(t *testing.T) {
	store, version := writeApprovalState(t)
	if err := PutPending(store, version, "support_leads"); err != nil {
		t.Fatalf("put pending: %v", err)
	}

	status, err := Reject(store, version.VersionID, "", "reviewer", "needs work")
	if err != nil {
		t.Fatalf("reject: %v", err)
	}
	if status.Status != "rejected" {
		t.Fatalf("unexpected status: %+v", status)
	}
	snapshot, err := store.ReadState()
	if err != nil {
		t.Fatal(err)
	}
	pointer := snapshot.CurrentArtifacts[version.ArtifactID]
	if pointer.ApprovalStatus != "rejected" || pointer.Confidence != "experimental" {
		t.Fatalf("pointer was not rejected: %+v", pointer)
	}
}

func writeApprovalState(t *testing.T) (state.Store, state.ArtifactVersion) {
	t.Helper()
	store := state.Open(filepath.Join(t.TempDir(), ".fbt", "state"))
	size := int64(10)
	version := state.ArtifactVersion{
		VersionID:      "artifact_version.knowledge_ops.case_summaries.sha256_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		ArtifactID:     "artifact.knowledge_ops.case_summaries",
		LogicalPath:    "target/artifacts/support/case_summaries/index.md",
		StoragePath:    "target/artifacts/support/case_summaries/index.md",
		Confidence:     "structural",
		ApprovalStatus: "pending",
		Descriptor: artifact.Descriptor{
			Digest:       "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			Size:         &size,
			ArtifactType: "fbt.artifact.markdown_document.v1",
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
				CurrentDigest:    version.Descriptor.Digest,
				LogicalPath:      version.LogicalPath,
				Confidence:       "structural",
				ApprovalStatus:   "pending",
			},
		},
		LatestRuns: map[string]state.LatestRun{},
	}); err != nil {
		t.Fatal(err)
	}
	return store, version
}
