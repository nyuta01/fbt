package state

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nyuta01/fbt/internal/artifact"
	"github.com/nyuta01/fbt/internal/manifest"
)

const (
	StateSchemaVersion             = "https://schemas.fbt.dev/fbt/state/v1.json"
	ArtifactVersionsSchemaVersion  = "https://schemas.fbt.dev/fbt/artifact-versions/v1.json"
	EvaluationResultsSchemaVersion = "https://schemas.fbt.dev/fbt/evaluation-results/v1.json"
	PolicyDecisionsSchemaVersion   = "https://schemas.fbt.dev/fbt/policy-decisions/v1.json"
	ApprovalsSchemaVersion         = "https://schemas.fbt.dev/fbt/approvals/v1.json"
)

type Store struct {
	Dir string
}

type Metadata struct {
	FBTSchemaVersion string `json:"fbt_schema_version"`
	FBTVersion       string `json:"fbt_version,omitempty"`
	ProjectName      string `json:"project_name"`
	ProjectID        string `json:"project_id,omitempty"`
	UpdatedAt        string `json:"updated_at,omitempty"`
	LastInvocationID string `json:"last_invocation_id,omitempty"`
}

type Snapshot struct {
	Metadata         Metadata                   `json:"metadata"`
	CurrentArtifacts map[string]ArtifactPointer `json:"current_artifacts"`
	LatestRuns       map[string]LatestRun       `json:"latest_runs"`
	PreviousManifest map[string]string          `json:"previous_manifest,omitempty"`
}

type ArtifactPointer struct {
	ArtifactID       string `json:"artifact_id"`
	CurrentVersionID string `json:"current_version_id"`
	CurrentDigest    string `json:"current_digest"`
	LogicalPath      string `json:"logical_path"`
	Confidence       string `json:"confidence,omitempty"`
	ApprovalStatus   string `json:"approval_status,omitempty"`
	CommittedAt      string `json:"committed_at,omitempty"`
	GeneratedBy      string `json:"generated_by,omitempty"`
}

type LatestRun struct {
	LatestRunID                string `json:"latest_run_id"`
	LatestSuccessfulRunID      string `json:"latest_successful_run_id,omitempty"`
	LatestStatus               string `json:"latest_status"`
	LatestEffectiveFingerprint string `json:"latest_effective_fingerprint,omitempty"`
}

type ArtifactVersionsIndex struct {
	Metadata         Metadata                   `json:"metadata"`
	ArtifactVersions map[string]ArtifactVersion `json:"artifact_versions"`
}

type ArtifactVersion struct {
	VersionID          string              `json:"version_id"`
	ArtifactID         string              `json:"artifact_id"`
	LogicalPath        string              `json:"logical_path"`
	StoragePath        string              `json:"storage_path"`
	Descriptor         artifact.Descriptor `json:"descriptor"`
	SemanticDescriptor map[string]any      `json:"semantic_descriptor,omitempty"`
	GeneratedBy        string              `json:"generated_by,omitempty"`
	Confidence         string              `json:"confidence,omitempty"`
	ApprovalStatus     string              `json:"approval_status,omitempty"`
	CreatedAt          string              `json:"created_at,omitempty"`
	CommittedAt        string              `json:"committed_at,omitempty"`
	Materials          []Material          `json:"materials,omitempty"`
}

type Material struct {
	ResourceID      string `json:"resource_id"`
	ArtifactVersion string `json:"artifact_version,omitempty"`
	Digest          string `json:"digest,omitempty"`
}

type ApprovalIndex struct {
	Metadata  Metadata            `json:"metadata"`
	Approvals map[string]Approval `json:"approvals"`
}

type Approval struct {
	ArtifactVersionID string  `json:"artifact_version_id"`
	ArtifactID        string  `json:"artifact_id"`
	Digest            string  `json:"digest"`
	Status            string  `json:"status"`
	ReviewGroup       string  `json:"review_group,omitempty"`
	Reviewer          string  `json:"reviewer,omitempty"`
	ApprovedAt        string  `json:"approved_at,omitempty"`
	ExpiresAt         *string `json:"expires_at"`
	Comment           string  `json:"comment,omitempty"`
	SupersededBy      *string `json:"superseded_by"`
}

type EvaluationResultsIndex struct {
	Metadata          Metadata                    `json:"metadata"`
	EvaluationResults map[string]EvaluationResult `json:"evaluation_results"`
}

type EvaluationResult struct {
	ResultID          string   `json:"result_id"`
	EvalID            string   `json:"eval_id"`
	ArtifactVersionID string   `json:"artifact_version_id"`
	TransformRunID    string   `json:"transform_run_id"`
	Status            string   `json:"status"`
	Score             *float64 `json:"score,omitempty"`
	Threshold         *float64 `json:"threshold,omitempty"`
	GrantsConfidence  string   `json:"grants_confidence,omitempty"`
	Runner            string   `json:"runner,omitempty"`
	DetailsPath       string   `json:"details_path,omitempty"`
}

type PolicyDecisionsIndex struct {
	Metadata        Metadata                  `json:"metadata"`
	PolicyDecisions map[string]PolicyDecision `json:"policy_decisions"`
}

type PolicyDecision struct {
	DecisionID        string        `json:"decision_id"`
	PolicyID          string        `json:"policy_id"`
	TransformID       string        `json:"transform_id"`
	TransformRunID    string        `json:"transform_run_id"`
	ArtifactVersionID string        `json:"artifact_version_id,omitempty"`
	Status            string        `json:"status"`
	Checks            []PolicyCheck `json:"checks,omitempty"`
	DecidedAt         string        `json:"decided_at,omitempty"`
}

type PolicyCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type LockInfo struct {
	InvocationID string `json:"invocation_id"`
	AcquiredAt   string `json:"acquired_at"`
	PID          int    `json:"pid"`
}

type Lock struct {
	store Store
	info  LockInfo
}

func Open(dir string) Store {
	return Store{Dir: dir}
}

func (s Store) Ensure() error {
	return os.MkdirAll(s.Dir, 0o755)
}

func (s Store) WriteManifest(m manifest.Manifest) error {
	data, err := m.JSON()
	if err != nil {
		return err
	}
	return s.atomicWriteBytes("manifest.json", append(data, '\n'))
}

func (s Store) WriteState(snapshot Snapshot) error {
	return s.atomicWriteJSON("state.json", snapshot)
}

func (s Store) ReadState() (Snapshot, error) {
	var snapshot Snapshot
	err := s.readJSON("state.json", &snapshot)
	if errors.Is(err, os.ErrNotExist) {
		return Snapshot{
			CurrentArtifacts: map[string]ArtifactPointer{},
			LatestRuns:       map[string]LatestRun{},
		}, nil
	}
	if err != nil {
		return Snapshot{}, err
	}
	if snapshot.CurrentArtifacts == nil {
		snapshot.CurrentArtifacts = map[string]ArtifactPointer{}
	}
	if snapshot.LatestRuns == nil {
		snapshot.LatestRuns = map[string]LatestRun{}
	}
	return snapshot, nil
}

func (s Store) ReadManifest() (manifest.Manifest, error) {
	var m manifest.Manifest
	err := s.readJSON("manifest.json", &m)
	if err != nil {
		return manifest.Manifest{}, err
	}
	return m, nil
}

func (s Store) WriteArtifactVersions(index ArtifactVersionsIndex) error {
	if index.ArtifactVersions == nil {
		index.ArtifactVersions = map[string]ArtifactVersion{}
	}
	return s.atomicWriteJSON("artifact_versions.json", index)
}

func (s Store) PutArtifactVersion(version ArtifactVersion) error {
	index, err := s.ReadArtifactVersions()
	if err != nil {
		return err
	}
	existing, ok := index.ArtifactVersions[version.VersionID]
	if ok {
		same, err := jsonEqual(existing, version)
		if err != nil {
			return err
		}
		if !same {
			return fmt.Errorf("artifact version %s is immutable", version.VersionID)
		}
		return nil
	}
	index.ArtifactVersions[version.VersionID] = version
	return s.WriteArtifactVersions(index)
}

func (s Store) ReadArtifactVersions() (ArtifactVersionsIndex, error) {
	var index ArtifactVersionsIndex
	err := s.readJSON("artifact_versions.json", &index)
	if errors.Is(err, os.ErrNotExist) {
		return ArtifactVersionsIndex{
			Metadata:         Metadata{FBTSchemaVersion: ArtifactVersionsSchemaVersion},
			ArtifactVersions: map[string]ArtifactVersion{},
		}, nil
	}
	if err != nil {
		return ArtifactVersionsIndex{}, err
	}
	if index.ArtifactVersions == nil {
		index.ArtifactVersions = map[string]ArtifactVersion{}
	}
	return index, nil
}

func (s Store) WriteApprovals(index ApprovalIndex) error {
	if index.Approvals == nil {
		index.Approvals = map[string]Approval{}
	}
	return s.atomicWriteJSON("approvals.json", index)
}

func (s Store) ReadApprovals() (ApprovalIndex, error) {
	var index ApprovalIndex
	err := s.readJSON("approvals.json", &index)
	if errors.Is(err, os.ErrNotExist) {
		return ApprovalIndex{
			Metadata:  Metadata{FBTSchemaVersion: ApprovalsSchemaVersion},
			Approvals: map[string]Approval{},
		}, nil
	}
	if err != nil {
		return ApprovalIndex{}, err
	}
	if index.Approvals == nil {
		index.Approvals = map[string]Approval{}
	}
	return index, nil
}

func (s Store) PutApproval(approval Approval) error {
	index, err := s.ReadApprovals()
	if err != nil {
		return err
	}
	index.Approvals[approval.ArtifactVersionID] = approval
	return s.WriteApprovals(index)
}

func (s Store) WriteEvaluationResults(index EvaluationResultsIndex) error {
	if index.EvaluationResults == nil {
		index.EvaluationResults = map[string]EvaluationResult{}
	}
	return s.atomicWriteJSON("evaluation_results.json", index)
}

func (s Store) ReadEvaluationResults() (EvaluationResultsIndex, error) {
	var index EvaluationResultsIndex
	err := s.readJSON("evaluation_results.json", &index)
	if errors.Is(err, os.ErrNotExist) {
		return EvaluationResultsIndex{
			Metadata:          Metadata{FBTSchemaVersion: EvaluationResultsSchemaVersion},
			EvaluationResults: map[string]EvaluationResult{},
		}, nil
	}
	if err != nil {
		return EvaluationResultsIndex{}, err
	}
	if index.EvaluationResults == nil {
		index.EvaluationResults = map[string]EvaluationResult{}
	}
	return index, nil
}

func (s Store) PutEvaluationResult(result EvaluationResult) error {
	index, err := s.ReadEvaluationResults()
	if err != nil {
		return err
	}
	index.EvaluationResults[result.ResultID] = result
	return s.WriteEvaluationResults(index)
}

func (s Store) WritePolicyDecisions(index PolicyDecisionsIndex) error {
	if index.PolicyDecisions == nil {
		index.PolicyDecisions = map[string]PolicyDecision{}
	}
	return s.atomicWriteJSON("policy_decisions.json", index)
}

func (s Store) AppendRunResult(record any) error {
	if err := s.Ensure(); err != nil {
		return err
	}
	path := filepath.Join(s.Dir, "run_results.jsonl")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return err
	}
	return file.Sync()
}

func (s Store) ReadRunResults() ([]map[string]any, error) {
	file, err := os.Open(filepath.Join(s.Dir, "run_results.jsonl"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var records []map[string]any
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, scanner.Err()
}

func (s Store) AcquireLock(invocationID string, staleAfter time.Duration) (*Lock, error) {
	if err := s.Ensure(); err != nil {
		return nil, err
	}
	lockPath := filepath.Join(s.Dir, ".lock")
	if staleAfter > 0 {
		if stale, err := s.lockIsStale(lockPath, staleAfter); err == nil && stale {
			if err := os.Remove(lockPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				return nil, err
			}
		}
	}
	info := LockInfo{
		InvocationID: invocationID,
		AcquiredAt:   time.Now().UTC().Format(time.RFC3339),
		PID:          os.Getpid(),
	}
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return nil, err
	}
	file, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("state lock already held: %s", lockPath)
		}
		return nil, err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		_ = file.Close()
		return nil, err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	return &Lock{store: s, info: info}, nil
}

func (s Store) lockIsStale(path string, staleAfter time.Duration) (bool, error) {
	var info LockInfo
	if err := readJSONPath(path, &info); err != nil {
		return false, err
	}
	acquiredAt, err := time.Parse(time.RFC3339, info.AcquiredAt)
	if err != nil {
		return false, err
	}
	return time.Since(acquiredAt) > staleAfter, nil
}

func (l *Lock) Release() error {
	path := filepath.Join(l.store.Dir, ".lock")
	var current LockInfo
	if err := readJSONPath(path, &current); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if current.InvocationID != l.info.InvocationID {
		return fmt.Errorf("cannot release lock held by %s", current.InvocationID)
	}
	return os.Remove(path)
}

func (s Store) atomicWriteJSON(name string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return s.atomicWriteBytes(name, append(data, '\n'))
}

func (s Store) atomicWriteBytes(name string, data []byte) error {
	if err := s.Ensure(); err != nil {
		return err
	}
	finalPath := filepath.Join(s.Dir, name)
	tmp, err := os.CreateTemp(s.Dir, "."+name+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, finalPath)
}

func (s Store) readJSON(name string, value any) error {
	return readJSONPath(filepath.Join(s.Dir, name), value)
}

func readJSONPath(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}

func jsonEqual(left, right any) (bool, error) {
	leftData, err := json.Marshal(left)
	if err != nil {
		return false, err
	}
	rightData, err := json.Marshal(right)
	if err != nil {
		return false, err
	}
	return string(leftData) == string(rightData), nil
}
