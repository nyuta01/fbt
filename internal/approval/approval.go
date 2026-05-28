package approval

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/nyuta01/fbt/internal/state"
)

type Status struct {
	ArtifactID        string          `json:"artifact_id"`
	ArtifactVersionID string          `json:"artifact_version_id"`
	Digest            string          `json:"digest"`
	LogicalPath       string          `json:"logical_path"`
	Status            string          `json:"status"`
	Confidence        string          `json:"confidence,omitempty"`
	ReviewGroup       string          `json:"review_group,omitempty"`
	Current           bool            `json:"current"`
	Approval          *state.Approval `json:"approval,omitempty"`
}

func PutPending(store state.Store, version state.ArtifactVersion, reviewGroup string) error {
	index, err := store.ReadApprovals()
	if err != nil {
		return err
	}
	if existing, ok := index.Approvals[version.VersionID]; ok && existing.Status != "" && existing.Status != "pending" {
		return nil
	}
	index.Approvals[version.VersionID] = state.Approval{
		ArtifactVersionID: version.VersionID,
		ArtifactID:        version.ArtifactID,
		Digest:            version.Descriptor.Digest,
		Status:            "pending",
		ReviewGroup:       reviewGroup,
	}
	return store.WriteApprovals(index)
}

func GetStatus(store state.Store, target, versionID string) (Status, error) {
	snapshot, version, current, pointer, err := resolve(store, target, versionID)
	if err != nil {
		return Status{}, err
	}
	_ = snapshot
	return statusFor(store, version, current, pointer)
}

func Approve(store state.Store, target, versionID, reviewer, comment string) (Status, error) {
	return putDecision(store, target, versionID, reviewer, comment, "approved")
}

func Reject(store state.Store, target, versionID, reviewer, comment string) (Status, error) {
	return putDecision(store, target, versionID, reviewer, comment, "rejected")
}

func putDecision(store state.Store, target, versionID, reviewer, comment, decision string) (Status, error) {
	snapshot, version, current, pointer, err := resolve(store, target, versionID)
	if err != nil {
		return Status{}, err
	}
	existing, _ := statusFor(store, version, current, pointer)
	approval := state.Approval{
		ArtifactVersionID: version.VersionID,
		ArtifactID:        version.ArtifactID,
		Digest:            version.Descriptor.Digest,
		Status:            decision,
		ReviewGroup:       existing.ReviewGroup,
		Reviewer:          reviewer,
		Comment:           comment,
	}
	if decision == "approved" {
		approval.ApprovedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if err := store.PutApproval(approval); err != nil {
		return Status{}, err
	}
	if current {
		pointer.ApprovalStatus = decision
		if decision == "approved" {
			pointer.Confidence = "reviewed"
		} else if decision == "rejected" {
			pointer.Confidence = "experimental"
		}
		snapshot.CurrentArtifacts[version.ArtifactID] = pointer
		if err := store.WriteState(snapshot); err != nil {
			return Status{}, err
		}
	}
	return statusFor(store, version, current, pointer)
}

func statusFor(store state.Store, version state.ArtifactVersion, current bool, pointer state.ArtifactPointer) (Status, error) {
	index, err := store.ReadApprovals()
	if err != nil {
		return Status{}, err
	}
	status := version.ApprovalStatus
	confidence := version.Confidence
	if current {
		if pointer.ApprovalStatus != "" {
			status = pointer.ApprovalStatus
		}
		if pointer.Confidence != "" {
			confidence = pointer.Confidence
		}
	}
	var approval *state.Approval
	reviewGroup := ""
	if existing, ok := index.Approvals[version.VersionID]; ok {
		value := existing
		approval = &value
		if value.Status != "" {
			status = value.Status
		}
		reviewGroup = value.ReviewGroup
	}
	if status == "" {
		status = "not_required"
	}
	return Status{
		ArtifactID:        version.ArtifactID,
		ArtifactVersionID: version.VersionID,
		Digest:            version.Descriptor.Digest,
		LogicalPath:       version.LogicalPath,
		Status:            status,
		Confidence:        confidence,
		ReviewGroup:       reviewGroup,
		Current:           current,
		Approval:          approval,
	}, nil
}

func resolve(store state.Store, target, versionID string) (state.Snapshot, state.ArtifactVersion, bool, state.ArtifactPointer, error) {
	snapshot, err := store.ReadState()
	if err != nil {
		return state.Snapshot{}, state.ArtifactVersion{}, false, state.ArtifactPointer{}, err
	}
	versions, err := store.ReadArtifactVersions()
	if err != nil {
		return state.Snapshot{}, state.ArtifactVersion{}, false, state.ArtifactPointer{}, err
	}
	if versionID != "" {
		version, ok := versions.ArtifactVersions[versionID]
		if !ok {
			return state.Snapshot{}, state.ArtifactVersion{}, false, state.ArtifactPointer{}, fmt.Errorf("artifact version not found: %s", versionID)
		}
		pointer := snapshot.CurrentArtifacts[version.ArtifactID]
		return snapshot, version, pointer.CurrentVersionID == version.VersionID, pointer, nil
	}
	if version, ok := versions.ArtifactVersions[target]; ok {
		pointer := snapshot.CurrentArtifacts[version.ArtifactID]
		return snapshot, version, pointer.CurrentVersionID == version.VersionID, pointer, nil
	}
	for artifactID, pointer := range snapshot.CurrentArtifacts {
		if matchesArtifact(artifactID, target) {
			version, ok := versions.ArtifactVersions[pointer.CurrentVersionID]
			if !ok {
				return state.Snapshot{}, state.ArtifactVersion{}, false, state.ArtifactPointer{}, fmt.Errorf("current artifact version not found: %s", pointer.CurrentVersionID)
			}
			return snapshot, version, true, pointer, nil
		}
	}
	matches := matchingVersions(versions, target)
	if len(matches) == 0 {
		return state.Snapshot{}, state.ArtifactVersion{}, false, state.ArtifactPointer{}, fmt.Errorf("artifact not found: %s", target)
	}
	version := matches[len(matches)-1]
	pointer := snapshot.CurrentArtifacts[version.ArtifactID]
	return snapshot, version, pointer.CurrentVersionID == version.VersionID, pointer, nil
}

func matchingVersions(index state.ArtifactVersionsIndex, target string) []state.ArtifactVersion {
	var matches []state.ArtifactVersion
	for _, version := range index.ArtifactVersions {
		if version.VersionID == target || matchesArtifact(version.ArtifactID, target) {
			matches = append(matches, version)
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].VersionID < matches[j].VersionID
	})
	return matches
}

func matchesArtifact(artifactID, target string) bool {
	return artifactID == target || strings.HasSuffix(artifactID, "."+target)
}
