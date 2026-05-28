package build

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nyuta01/fbt/internal/approval"
	"github.com/nyuta01/fbt/internal/artifact"
	evalmgr "github.com/nyuta01/fbt/internal/eval"
	"github.com/nyuta01/fbt/internal/manifest"
	"github.com/nyuta01/fbt/internal/parser"
	"github.com/nyuta01/fbt/internal/planner"
	"github.com/nyuta01/fbt/internal/policy"
	"github.com/nyuta01/fbt/internal/protocol"
	"github.com/nyuta01/fbt/internal/runner"
	"github.com/nyuta01/fbt/internal/security"
	"github.com/nyuta01/fbt/internal/state"
)

type Options struct {
	ProjectDir string
	StateDir   string
	Select     string
	FBTVersion string
}

type Result struct {
	InvocationID string
	Plan         planner.Plan
	Runs         []Run
}

type Run struct {
	TransformID       string
	TransformRunID    string
	Status            string
	CommittedVersions []string
	EvaluationResults []string
	Usage             map[string]any
	Provenance        map[string]any
}

type outputCandidate struct {
	Name         string `json:"name"`
	ArtifactType string `json:"artifact_type"`
	Path         string `json:"path"`
	DeclaredPath string `json:"declared_path"`
}

func RunBuild(ctx context.Context, options Options) (Result, error) {
	if options.FBTVersion == "" {
		options.FBTVersion = "0.0.0-dev"
	}
	parseResult, err := parser.ParseProject(parser.Options{ProjectDir: options.ProjectDir})
	if err != nil {
		return Result{}, err
	}
	currentManifest, err := manifest.Build(parseResult, manifest.BuildOptions{FBTVersion: options.FBTVersion})
	if err != nil {
		return Result{}, err
	}
	stateDir := options.StateDir
	if stateDir == "" {
		stateDir = filepath.Join(parseResult.ProjectDir, parseResult.Config.State.Path)
	}
	store := state.Open(stateDir)
	lock, err := store.AcquireLock(newID("inv"), 30*time.Minute)
	if err != nil {
		return Result{}, err
	}
	defer lock.Release()

	invocationID := newID("inv")
	result := Result{InvocationID: invocationID}
	var previous *manifest.Manifest
	if prev, err := store.ReadManifest(); err == nil {
		previous = &prev
	}
	snapshot, err := store.ReadState()
	if err != nil {
		return result, err
	}
	if err := store.WriteManifest(currentManifest); err != nil {
		return result, err
	}
	if err := store.AppendRunResult(map[string]any{
		"record_type":   "invocation_started",
		"invocation_id": invocationID,
		"started_at":    time.Now().UTC().Format(time.RFC3339),
		"command":       "build",
		"project_name":  parseResult.Config.Name,
		"target_name":   "local",
	}); err != nil {
		return result, err
	}

	selected, err := selectedTransformIDs(currentManifest, options.Select)
	if err != nil {
		return result, err
	}
	plan := planner.Build(planner.Inputs{Manifest: currentManifest, PreviousManifest: previous, State: snapshot, Selected: selected})
	result.Plan = plan

	for _, node := range plan.Nodes {
		if node.Action != planner.ActionRun {
			continue
		}
		run, err := executeTransform(ctx, parseResult, currentManifest, store, &snapshot, node.TransformID, invocationID)
		if err != nil {
			return result, err
		}
		result.Runs = append(result.Runs, run)
	}

	if err := store.WriteState(snapshot); err != nil {
		return result, err
	}
	status := "success"
	if plan.Summary.Blocked > 0 {
		status = "blocked"
	}
	if err := store.AppendRunResult(map[string]any{
		"record_type":   "invocation_completed",
		"invocation_id": invocationID,
		"completed_at":  time.Now().UTC().Format(time.RFC3339),
		"status":        status,
		"summary": map[string]any{
			"selected": plan.Summary.Selected,
			"success":  len(result.Runs),
			"blocked":  plan.Summary.Blocked,
			"skipped":  plan.Summary.Skipped,
		},
	}); err != nil {
		return result, err
	}
	return result, nil
}

func executeTransform(ctx context.Context, parseResult parser.Result, m manifest.Manifest, store state.Store, snapshot *state.Snapshot, transformID string, invocationID string) (Run, error) {
	transform := m.Transforms[transformID]
	runnerName := strings.TrimPrefix(transform.Runner, "runner."+parseResult.Config.Name+".")
	discovery := runner.NewDiscovery(parseResult.ProjectDir, parseResult.Config)
	resolved, err := discovery.Resolve(runnerName)
	if err != nil {
		return Run{}, err
	}
	client, err := runner.StartProtocolClient(ctx, resolved)
	if err != nil {
		return Run{}, err
	}
	defer client.Close()
	if _, err := client.Initialize(ctx, protocol.InitializeParams{
		Core: map[string]string{"name": "fbt-core", "version": "0.0.0-dev"},
		Protocol: map[string]any{
			"versions": []string{"0.1"},
			"framing":  "jsonl",
		},
		CapabilityRequest: []string{"run_transform", "stream_events", "output_candidates", "cancellation"},
	}); err != nil {
		return Run{}, err
	}
	policyResource := policyForTransform(m, transform)
	runCtx := ctx
	var cancel context.CancelFunc
	if policyResource != nil {
		if timeout := policy.Timeout(*policyResource); timeout > 0 {
			runCtx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}
	}

	transformRunID := newID("transform_run.run")
	workRootRel := filepath.Join(".fbt", "work", invocationID, transform.Name)
	workRoot := filepath.Join(parseResult.ProjectDir, workRootRel)
	workTemp := filepath.Join(workRoot, "tmp")
	workOutputs := filepath.Join(workRoot, "outputs")
	if err := os.MkdirAll(workTemp, 0o755); err != nil {
		return Run{}, err
	}
	if err := os.MkdirAll(workOutputs, 0o755); err != nil {
		return Run{}, err
	}

	outcome, err := client.RunTransform(runCtx, protocol.RunTransformParams{
		Mode:           "run",
		InvocationID:   invocationID,
		TransformRunID: transformRunID,
		Transform: map[string]any{
			"unique_id": transform.UniqueID,
			"name":      transform.Name,
			"type":      transform.TransformType,
		},
		Model:   transform.Model,
		Tools:   protocolTools(transform),
		Policy:  protocolPolicy(policyResource),
		Outputs: protocolOutputs(transform),
		Work: map[string]any{
			"root":    workRoot,
			"temp":    workTemp,
			"outputs": workOutputs,
		},
	})
	if err != nil {
		return Run{}, err
	}
	candidates, err := decodeOutputCandidates(outcome)
	if err != nil {
		return Run{}, err
	}
	declared := declaredOutputs(transform)
	run := Run{
		TransformID:    transformID,
		TransformRunID: transformRunID,
		Status:         outcome.Result.Status,
		Usage:          outcome.Result.Usage,
		Provenance:     outcome.Result.Provenance,
	}
	for _, candidate := range candidates {
		if err := security.RequireWithin(workOutputs, candidate.Path); err != nil {
			return Run{}, fmt.Errorf("output candidate outside work outputs: %w", err)
		}
		output, ok := declared[candidate.Name]
		if !ok {
			return Run{}, fmt.Errorf("undeclared output candidate: %s", candidate.Name)
		}
		relCandidate, err := filepath.Rel(parseResult.ProjectDir, candidate.Path)
		if err != nil {
			return Run{}, err
		}
		descriptor, err := artifact.Describe(parseResult.ProjectDir, relCandidate, output.ArtifactType)
		if err != nil {
			return Run{}, err
		}
		decision := policy.EvaluateCommit(parseResult.ProjectDir, parseResult.Config.ArtifactPath, policyResource, output, descriptor)
		if decision.Status != "allowed" {
			return Run{}, fmt.Errorf("policy denied output %s: %s", output.Name, firstFailedCheck(decision))
		}
		versionID, err := artifact.VersionID(parseResult.Config.Name, output.Name, descriptor.Digest)
		if err != nil {
			return Run{}, err
		}
		evalOutcome, evalErr := evalmgr.RunForCandidate(parseResult.ProjectDir, transform, m.Evals, versionID, transformRunID, candidate.Path)
		for _, result := range evalOutcome.Results {
			if err := store.PutEvaluationResult(result); err != nil {
				return Run{}, err
			}
			run.EvaluationResults = append(run.EvaluationResults, result.ResultID)
		}
		if evalErr != nil {
			return Run{}, evalErr
		}
		logicalAbs := filepath.Join(parseResult.ProjectDir, output.DeclaredPath)
		if err := commitPath(candidate.Path, logicalAbs); err != nil {
			return Run{}, err
		}
		review := reviewForTransform(transform, policyResource)
		approvalStatus := "not_required"
		confidence := "structural"
		if evalOutcome.Confidence != "" {
			confidence = maxConfidence(confidence, evalOutcome.Confidence)
		}
		if review.Required {
			approvalStatus = "pending"
			if evalOutcome.Confidence == "" {
				confidence = "experimental"
			}
		}
		version := state.ArtifactVersion{
			VersionID:      versionID,
			ArtifactID:     output.UniqueID,
			LogicalPath:    output.DeclaredPath,
			StoragePath:    output.DeclaredPath,
			Descriptor:     descriptor,
			GeneratedBy:    transformRunID,
			Confidence:     confidence,
			ApprovalStatus: approvalStatus,
			CreatedAt:      time.Now().UTC().Format(time.RFC3339),
			CommittedAt:    time.Now().UTC().Format(time.RFC3339),
		}
		if err := store.PutArtifactVersion(version); err != nil {
			return Run{}, err
		}
		if review.Required {
			if err := approval.PutPending(store, version, review.Group); err != nil {
				return Run{}, err
			}
		}
		snapshot.CurrentArtifacts[output.UniqueID] = state.ArtifactPointer{
			ArtifactID:       output.UniqueID,
			CurrentVersionID: versionID,
			CurrentDigest:    descriptor.Digest,
			LogicalPath:      output.DeclaredPath,
			Confidence:       confidence,
			ApprovalStatus:   approvalStatus,
			CommittedAt:      version.CommittedAt,
			GeneratedBy:      transformRunID,
		}
		run.CommittedVersions = append(run.CommittedVersions, versionID)
	}
	snapshot.LatestRuns[transformID] = state.LatestRun{
		LatestRunID:                transformRunID,
		LatestSuccessfulRunID:      transformRunID,
		LatestStatus:               outcome.Result.Status,
		LatestEffectiveFingerprint: transform.Fingerprint["effective"],
	}
	if err := store.AppendRunResult(map[string]any{
		"record_type":        "transform_run",
		"invocation_id":      invocationID,
		"run_id":             transformRunID,
		"transform_id":       transformID,
		"status":             outcome.Result.Status,
		"committed_versions": run.CommittedVersions,
		"evaluation_results": run.EvaluationResults,
		"usage":              run.Usage,
		"provenance":         run.Provenance,
	}); err != nil {
		return Run{}, err
	}
	return run, nil
}

func policyForTransform(m manifest.Manifest, transform manifest.TransformResource) *manifest.PolicyResource {
	if transform.Policy == "" {
		return nil
	}
	policyResource, ok := m.Policies[transform.Policy]
	if !ok {
		return nil
	}
	return &policyResource
}

func firstFailedCheck(decision policy.Decision) string {
	for _, check := range decision.Checks {
		if check.Status == "fail" {
			return check.Message
		}
	}
	return "unknown policy denial"
}

func protocolOutputs(transform manifest.TransformResource) []any {
	outputs := make([]any, 0, len(transform.Outputs))
	for _, output := range transform.Outputs {
		outputs = append(outputs, map[string]any{
			"name":          output.Name,
			"artifact_type": output.ArtifactType,
			"declared_path": output.DeclaredPath,
		})
	}
	return outputs
}

func protocolTools(transform manifest.TransformResource) []any {
	tools := make([]any, 0, len(transform.Tools))
	for _, tool := range transform.Tools {
		tools = append(tools, map[string]any{"name": tool})
	}
	return tools
}

func protocolPolicy(policyResource *manifest.PolicyResource) map[string]any {
	if policyResource == nil {
		return nil
	}
	return map[string]any{
		"unique_id":     policyResource.UniqueID,
		"name":          policyResource.Name,
		"read_scope":    policyResource.ReadScope,
		"write_scope":   policyResource.WriteScope,
		"network":       policyResource.Network,
		"tools":         policyResource.Tools,
		"limits":        policyResource.Limits,
		"review":        policyResource.Review,
		"fingerprint":   policyResource.Fingerprint,
		"resource_type": policyResource.ResourceType,
	}
}

func declaredOutputs(transform manifest.TransformResource) map[string]manifest.TransformOutput {
	outputs := map[string]manifest.TransformOutput{}
	for _, output := range transform.Outputs {
		outputs[output.Name] = output
	}
	return outputs
}

func decodeOutputCandidates(outcome protocol.RunOutcome) ([]outputCandidate, error) {
	var candidates []outputCandidate
	for _, notification := range outcome.OutputCandidates {
		for _, raw := range notification.Outputs {
			data, err := json.Marshal(raw)
			if err != nil {
				return nil, err
			}
			var candidate outputCandidate
			if err := json.Unmarshal(data, &candidate); err != nil {
				return nil, err
			}
			candidates = append(candidates, candidate)
		}
	}
	if len(candidates) == 0 {
		for _, raw := range outcome.Result.Outputs {
			data, err := json.Marshal(raw)
			if err != nil {
				return nil, err
			}
			var candidate outputCandidate
			if err := json.Unmarshal(data, &candidate); err != nil {
				return nil, err
			}
			candidates = append(candidates, candidate)
		}
	}
	return candidates, nil
}

func selectedTransformIDs(m manifest.Manifest, expr string) (map[string]struct{}, error) {
	if expr == "" {
		return nil, nil
	}
	selected := map[string]struct{}{}
	for id, transform := range m.Transforms {
		if transform.Name == strings.Trim(expr, "+") || id == expr {
			selected[id] = struct{}{}
		}
	}
	return selected, nil
}

type reviewRequirement struct {
	Required bool
	Group    string
}

func reviewForTransform(transform manifest.TransformResource, policyResource *manifest.PolicyResource) reviewRequirement {
	var review reviewRequirement
	mergeReview(&review, transform.Review)
	if policyResource != nil {
		mergeReview(&review, policyResource.Review)
	}
	required, _ := transform.Cache["review_required"].(bool)
	if required {
		review.Required = true
	}
	value, ok := transform.Fingerprint["review_required"]
	if ok && value == "true" {
		review.Required = true
	}
	return review
}

func mergeReview(target *reviewRequirement, values map[string]any) {
	if len(values) == 0 {
		return
	}
	if required, ok := values["required"].(bool); ok && required {
		target.Required = true
	}
	if group, ok := values["group"].(string); ok && group != "" {
		target.Group = group
	}
}

func maxConfidence(left, right string) string {
	order := map[string]int{
		"experimental": 0,
		"structural":   1,
		"semantic":     2,
		"exact":        3,
		"reviewed":     4,
	}
	if left == "" {
		return right
	}
	if right == "" {
		return left
	}
	if order[right] > order[left] {
		return right
	}
	return left
}

func commitPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("cannot commit symlink: %s", path)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func newID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UTC().UnixNano())
}
