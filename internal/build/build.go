package build

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nyuta01/fbt/internal/artifact"
	evalmgr "github.com/nyuta01/fbt/internal/eval"
	"github.com/nyuta01/fbt/internal/graph"
	"github.com/nyuta01/fbt/internal/manifest"
	"github.com/nyuta01/fbt/internal/parser"
	"github.com/nyuta01/fbt/internal/planner"
	"github.com/nyuta01/fbt/internal/policy"
	"github.com/nyuta01/fbt/internal/protocol"
	"github.com/nyuta01/fbt/internal/runner"
	"github.com/nyuta01/fbt/internal/security"
	"github.com/nyuta01/fbt/internal/state"
	versioninfo "github.com/nyuta01/fbt/internal/version"
)

type Options struct {
	ProjectDir string
	StateDir   string
	Select     string
	Force      bool
	FBTVersion string
}

type Result struct {
	InvocationID string
	Plan         planner.Plan
	Runs         []Run
}

type Run struct {
	TransformID        string
	TransformRunID     string
	Status             string
	StartedAt          string
	CompletedAt        string
	CommittedVersions  []string
	CommittedArtifacts []CommittedArtifact
	EvaluationResults  []string
	EvaluationDetails  []state.EvaluationResult
	PolicyDecisions    []string
	Usage              map[string]any
	Provenance         map[string]any
	Events             []protocol.Event
}

type CommittedArtifact struct {
	ArtifactID   string `json:"artifact_id"`
	VersionID    string `json:"version_id"`
	LogicalPath  string `json:"logical_path"`
	StoragePath  string `json:"storage_path"`
	ArtifactType string `json:"artifact_type"`
}

type outputCandidate struct {
	Name         string `json:"name"`
	ArtifactType string `json:"artifact_type"`
	Path         string `json:"path"`
	DeclaredPath string `json:"declared_path"`
}

func RunBuild(ctx context.Context, options Options) (result Result, err error) {
	if options.FBTVersion == "" {
		options.FBTVersion = versioninfo.Version
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
	result = Result{InvocationID: invocationID}
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
	invocationStartedAt := time.Now().UTC()
	if err := store.AppendRunResult(map[string]any{
		"record_type":   "invocation_started",
		"invocation_id": invocationID,
		"started_at":    invocationStartedAt.Format(time.RFC3339),
		"command":       "build",
		"project_name":  parseResult.Config.Name,
		"target_name":   "local",
	}); err != nil {
		return result, err
	}
	invocationStarted := true
	stateWritten := false
	defer func() {
		if !invocationStarted {
			return
		}
		if !stateWritten && len(result.Runs) > 0 {
			if writeErr := store.WriteState(snapshot); writeErr != nil && err == nil {
				err = writeErr
			}
		}
		completedAt := time.Now().UTC()
		completionErr := appendInvocationCompleted(store, invocationID, invocationStatus(err, result.Plan), result, completedAt)
		if completionErr != nil && err == nil {
			err = completionErr
		}
	}()

	selected, err := selectedTransformIDs(currentManifest, options.Select)
	if err != nil {
		return result, err
	}
	plan := planner.Build(planner.Inputs{Manifest: currentManifest, PreviousManifest: previous, State: snapshot, Selected: selected, Force: options.Force})
	result.Plan = plan

	for i := range result.Plan.Nodes {
		node := &result.Plan.Nodes[i]
		if node.Action != planner.ActionRun {
			continue
		}
		transform := currentManifest.Transforms[node.TransformID]
		if blockedReasons, nextSteps := planner.RuntimeBlock(transform, snapshot); len(blockedReasons) > 0 {
			node.Action = planner.ActionBlocked
			node.DirtyReasons = nil
			node.BlockedReasons = blockedReasons
			node.NextSteps = nextSteps
			result.Plan.Summary.Run--
			result.Plan.Summary.Blocked++
			continue
		}
		run, runErr := executeTransform(ctx, parseResult, currentManifest, store, &snapshot, *node, invocationID)
		if run.TransformRunID != "" {
			result.Runs = append(result.Runs, run)
		}
		if runErr != nil {
			return result, runErr
		}
	}

	if err := store.WriteState(snapshot); err != nil {
		return result, err
	}
	stateWritten = true
	return result, nil
}

func executeTransform(ctx context.Context, parseResult parser.Result, m manifest.Manifest, store state.Store, snapshot *state.Snapshot, node planner.Node, invocationID string) (run Run, err error) {
	transformID := node.TransformID
	transform := m.Transforms[transformID]
	transformRunID := newID("transform_run.run")
	startedAt := time.Now().UTC()
	run = Run{
		TransformID:    transformID,
		TransformRunID: transformRunID,
		Status:         "started",
		StartedAt:      startedAt.Format(time.RFC3339Nano),
	}
	receiptWritten := false
	var secretNames []string
	defer func() {
		if err == nil || receiptWritten {
			return
		}
		if run.Status == "" || run.Status == "started" || run.Status == "success" {
			run.Status = failureStatus(err)
		}
		if run.CompletedAt == "" {
			run.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
		}
		updateLatestRun(snapshot, transformID, transform, run.TransformRunID, run.Status, false)
		if receiptErr := appendTransformRunResult(store, invocationID, run, startedAt, err, secretNames); receiptErr != nil {
			err = fmt.Errorf("%w; additionally failed to persist transform run receipt: %v", err, receiptErr)
		}
	}()

	runnerName := strings.TrimPrefix(transform.Runner, "runner."+parseResult.Config.Name+".")
	discovery := runner.NewDiscovery(parseResult.ProjectDir, parseResult.Config)
	resolved, err := discovery.Resolve(runnerName)
	if err != nil {
		return run, err
	}
	secretNames = append(secretNames, resolved.Env...)
	client, err := runner.StartProtocolClient(ctx, resolved)
	if err != nil {
		return run, err
	}
	defer client.Close()
	initResult, err := client.Initialize(ctx, protocol.InitializeParams{
		Core: map[string]string{"name": "fbt-core", "version": m.Metadata.FBTVersion},
		Protocol: map[string]any{
			"versions": []string{"0.1"},
			"framing":  "jsonl",
		},
		CapabilityRequest: []string{"run_transform", "stream_events", "output_candidates", "cancellation"},
	})
	if err != nil {
		return run, err
	}
	if diagnostics := runner.ValidateCapabilities(initResult, []runner.CapabilityRequirement{capabilityRequirement(transformID, transform)}); runner.HasErrors(diagnostics) {
		return run, runner.CapabilityError(diagnostics)
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

	workRootRel := filepath.Join(".fbt", "work", invocationID, transform.Name)
	workRoot := filepath.Join(parseResult.ProjectDir, workRootRel)
	workTemp := filepath.Join(workRoot, "tmp")
	workOutputs := filepath.Join(workRoot, "outputs")
	if err := os.MkdirAll(workTemp, 0o755); err != nil {
		return run, err
	}
	if err := os.MkdirAll(workOutputs, 0o755); err != nil {
		return run, err
	}
	artifactVersions, err := store.ReadArtifactVersions()
	if err != nil {
		return run, err
	}

	transformPayload := map[string]any{
		"unique_id": transform.UniqueID,
		"name":      transform.Name,
		"type":      transform.TransformType,
	}
	if len(transform.Command) > 0 {
		transformPayload["command"] = transform.Command
	}
	outcome, err := client.RunTransform(runCtx, protocol.RunTransformParams{
		Mode:           "run",
		InvocationID:   invocationID,
		TransformRunID: transformRunID,
		Transform:      transformPayload,
		Runner:         protocolRunner(transform, m, resolved),
		Inputs:         protocolInputs(parseResult.ProjectDir, transform, m, *snapshot, artifactVersions),
		Model:          transform.Model,
		Tools:          protocolTools(transform),
		Policy:         protocolPolicy(policyResource),
		Outputs:        protocolOutputs(transform),
		Assets:         protocolAssets(parseResult.ProjectDir, transform, m),
		State:          protocolState(transformID, transform, *snapshot, artifactVersions, node),
		Work: map[string]any{
			"root":    workRoot,
			"temp":    workTemp,
			"outputs": workOutputs,
		},
	})
	run.Events = redactedProtocolEvents(outcome.Events)
	if err != nil {
		return run, err
	}
	run.Status = outcome.Result.Status
	if run.Status == "" {
		run.Status = "success"
	}
	run.Usage = outcome.Result.Usage
	run.Provenance = outcome.Result.Provenance
	if run.Status != "success" {
		return run, fmt.Errorf("runner returned status %s", run.Status)
	}
	candidates, err := decodeOutputCandidates(outcome)
	if err != nil {
		return run, err
	}
	declared := declaredOutputs(transform)
	for _, candidate := range candidates {
		if err := security.RequireWithin(workOutputs, candidate.Path); err != nil {
			return run, fmt.Errorf("output candidate outside work outputs: %w", err)
		}
		output, ok := declared[candidate.Name]
		if !ok {
			return run, fmt.Errorf("undeclared output candidate: %s", candidate.Name)
		}
		relCandidate, err := filepath.Rel(parseResult.ProjectDir, candidate.Path)
		if err != nil {
			return run, err
		}
		descriptor, err := artifact.Describe(parseResult.ProjectDir, relCandidate, output.ArtifactType)
		if err != nil {
			return run, err
		}
		semanticDescriptor, err := artifact.SemanticDescriptor(parseResult.ProjectDir, relCandidate, output.ArtifactType)
		if err != nil {
			return run, err
		}
		versionID, err := artifact.VersionID(parseResult.Config.Name, output.Name, descriptor.Digest)
		if err != nil {
			return run, err
		}
		decision := policy.EvaluateCommit(parseResult.ProjectDir, parseResult.Config.ArtifactPath, policyResource, output, descriptor)
		policyDecision := buildPolicyDecision(parseResult.Config.Name, transformID, transformRunID, versionID, output.Name, policyResource, decision)
		if err := store.PutPolicyDecision(policyDecision); err != nil {
			return run, err
		}
		run.PolicyDecisions = append(run.PolicyDecisions, policyDecision.DecisionID)
		if decision.Status != "allowed" {
			run.Status = "policy_denied"
			return run, fmt.Errorf("policy denied output %s: %s", output.Name, firstFailedCheck(decision))
		}
		evalOutcome, evalErr := evalmgr.RunForCandidate(parseResult.ProjectDir, transform, m.Evals, versionID, transformRunID, candidate.Path)
		for _, result := range evalOutcome.Results {
			if err := store.PutEvaluationResult(result); err != nil {
				return run, err
			}
			run.EvaluationResults = append(run.EvaluationResults, result.ResultID)
			run.EvaluationDetails = append(run.EvaluationDetails, result)
		}
		if evalErr != nil {
			run.Status = "eval_failed"
			return run, evalErr
		}
		versionStorageRel := filepath.ToSlash(filepath.Join(".fbt", "artifacts", versionID, "content"))
		versionStorageAbs := filepath.Join(parseResult.ProjectDir, versionStorageRel)
		artifactVersions, err := store.ReadArtifactVersions()
		if err != nil {
			return run, err
		}
		existingVersion, versionExists := artifactVersions.ArtifactVersions[versionID]
		if versionExists {
			versionStorageRel = existingVersion.StoragePath
		} else {
			if err := commitPath(candidate.Path, versionStorageAbs); err != nil {
				return run, err
			}
		}
		logicalAbs := filepath.Join(parseResult.ProjectDir, output.DeclaredPath)
		if err := commitPath(candidate.Path, logicalAbs); err != nil {
			return run, err
		}
		confidence := "structural"
		if evalOutcome.Confidence != "" {
			confidence = maxConfidence(confidence, evalOutcome.Confidence)
		}
		version := state.ArtifactVersion{
			VersionID:          versionID,
			ArtifactID:         output.UniqueID,
			LogicalPath:        output.DeclaredPath,
			StoragePath:        versionStorageRel,
			Descriptor:         descriptor,
			SemanticDescriptor: semanticDescriptor,
			GeneratedBy:        transformRunID,
			Confidence:         confidence,
			CreatedAt:          time.Now().UTC().Format(time.RFC3339),
			CommittedAt:        time.Now().UTC().Format(time.RFC3339),
		}
		if versionExists {
			version = existingVersion
		} else {
			if err := store.PutArtifactVersion(version); err != nil {
				return run, err
			}
		}
		snapshot.CurrentArtifacts[output.UniqueID] = state.ArtifactPointer{
			ArtifactID:       output.UniqueID,
			CurrentVersionID: versionID,
			CurrentDigest:    descriptor.Digest,
			LogicalPath:      output.DeclaredPath,
			Confidence:       confidence,
			CommittedAt:      version.CommittedAt,
			GeneratedBy:      transformRunID,
		}
		run.CommittedVersions = append(run.CommittedVersions, versionID)
		run.CommittedArtifacts = append(run.CommittedArtifacts, CommittedArtifact{
			ArtifactID:   output.UniqueID,
			VersionID:    versionID,
			LogicalPath:  output.DeclaredPath,
			StoragePath:  version.StoragePath,
			ArtifactType: output.ArtifactType,
		})
	}
	updateLatestRun(snapshot, transformID, transform, transformRunID, run.Status, true)
	completedAt := time.Now().UTC()
	run.CompletedAt = completedAt.Format(time.RFC3339Nano)
	receiptWritten = true
	if err := appendTransformRunResult(store, invocationID, run, startedAt, nil, nil); err != nil {
		return run, err
	}
	return run, nil
}

func appendInvocationCompleted(store state.Store, invocationID, status string, result Result, completedAt time.Time) error {
	return store.AppendRunResult(map[string]any{
		"record_type":   "invocation_completed",
		"invocation_id": invocationID,
		"completed_at":  completedAt.Format(time.RFC3339),
		"status":        status,
		"summary": map[string]any{
			"selected": result.Plan.Summary.Selected,
			"success":  countRuns(result.Runs, "success"),
			"failed":   countFailedRuns(result.Runs),
			"blocked":  result.Plan.Summary.Blocked,
			"skipped":  result.Plan.Summary.Skipped,
		},
	})
}

func appendTransformRunResult(store state.Store, invocationID string, run Run, startedAt time.Time, runErr error, secretNames []string) error {
	completedAt := run.CompletedAt
	if completedAt == "" {
		completedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	record := map[string]any{
		"record_type":         "transform_run",
		"invocation_id":       invocationID,
		"run_id":              run.TransformRunID,
		"transform_id":        run.TransformID,
		"status":              run.Status,
		"started_at":          run.StartedAt,
		"completed_at":        completedAt,
		"duration_ms":         durationMillis(startedAt, completedAt),
		"committed_versions":  run.CommittedVersions,
		"committed_artifacts": run.CommittedArtifacts,
		"evaluation_results":  run.EvaluationResults,
		"policy_decisions":    run.PolicyDecisions,
		"usage":               run.Usage,
		"provenance":          run.Provenance,
		"events":              run.Events,
	}
	if runErr != nil {
		record["error"] = safeErrorRecord(runErr, run.Status, secretNames)
	}
	if len(run.EvaluationDetails) > 0 {
		record["evaluation_summaries"] = evaluationSummaries(run.EvaluationDetails)
	}
	return store.AppendRunResult(record)
}

func evaluationSummaries(results []state.EvaluationResult) []map[string]any {
	summaries := make([]map[string]any, 0, len(results))
	for _, result := range results {
		summary := map[string]any{
			"result_id": result.ResultID,
			"eval_id":   result.EvalID,
			"status":    result.Status,
		}
		if result.Reason != "" {
			summary["reason"] = result.Reason
		}
		if result.Hint != "" {
			summary["hint"] = result.Hint
		}
		if result.GrantsConfidence != "" {
			summary["grants_confidence"] = result.GrantsConfidence
		}
		summaries = append(summaries, summary)
	}
	return summaries
}

func updateLatestRun(snapshot *state.Snapshot, transformID string, transform manifest.TransformResource, runID, status string, successful bool) {
	previous := snapshot.LatestRuns[transformID]
	latest := state.LatestRun{
		LatestRunID:                runID,
		LatestSuccessfulRunID:      previous.LatestSuccessfulRunID,
		LatestStatus:               status,
		LatestEffectiveFingerprint: previous.LatestEffectiveFingerprint,
	}
	if successful {
		latest.LatestSuccessfulRunID = runID
		latest.LatestEffectiveFingerprint = transform.Fingerprint["effective"]
	}
	snapshot.LatestRuns[transformID] = latest
}

func invocationStatus(err error, plan planner.Plan) string {
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return "cancelled"
		}
		return "failed"
	}
	if plan.Summary.Blocked > 0 {
		return "blocked"
	}
	return "success"
}

func failureStatus(err error) string {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "cancelled"
	}
	return "failed"
}

func countRuns(runs []Run, status string) int {
	var count int
	for _, run := range runs {
		if run.Status == status {
			count++
		}
	}
	return count
}

func countFailedRuns(runs []Run) int {
	var count int
	for _, run := range runs {
		if run.Status != "" && run.Status != "success" && run.Status != "blocked" && run.Status != "skipped" {
			count++
		}
	}
	return count
}

func durationMillis(startedAt time.Time, completedAt string) int64 {
	completed, err := time.Parse(time.RFC3339Nano, completedAt)
	if err != nil {
		completed, err = time.Parse(time.RFC3339, completedAt)
	}
	if err != nil {
		return 0
	}
	return completed.Sub(startedAt).Milliseconds()
}

func safeErrorRecord(err error, status string, secretNames []string) map[string]any {
	message := err.Error()
	if len(secretNames) > 0 {
		message = policy.RedactSecrets(message, envSecretValues(secretNames))
	}
	return map[string]any{
		"kind":    errorKind(err, status),
		"message": message,
	}
}

func errorKind(err error, status string) string {
	switch status {
	case "policy_denied":
		return "policy_denied"
	case "eval_failed":
		return "eval_failed"
	case "cancelled":
		return "cancelled"
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "cancelled"
	}
	if errors.Is(err, runner.ErrCapabilityIncompatible) {
		return "runner_capability_incompatible"
	}
	var rpcErr protocol.JSONRPCError
	if errors.As(err, &rpcErr) {
		return "runner_protocol_error"
	}
	var processErr protocol.RunnerProcessError
	if errors.As(err, &processErr) {
		return "runner_protocol_error"
	}
	message := err.Error()
	switch {
	case strings.Contains(message, "output candidate outside work outputs"):
		return "runner_contract_violation"
	case strings.Contains(message, "undeclared output candidate"):
		return "runner_contract_violation"
	case strings.Contains(message, "policy denied"):
		return "policy_denied"
	case strings.Contains(message, "eval failed"):
		return "eval_failed"
	default:
		return "failed"
	}
}

func envSecretValues(names []string) map[string]string {
	values := map[string]string{}
	for _, name := range names {
		if value, ok := os.LookupEnv(name); ok {
			values[name] = value
		}
	}
	return values
}

func capabilityRequirement(transformID string, transform manifest.TransformResource) runner.CapabilityRequirement {
	artifactTypes := make([]string, 0, len(transform.Outputs))
	for _, output := range transform.Outputs {
		artifactTypes = append(artifactTypes, output.ArtifactType)
	}
	return runner.CapabilityRequirement{
		TransformID:             transformID,
		TransformType:           transform.TransformType,
		ArtifactTypes:           artifactTypes,
		RequireOutputCandidates: true,
	}
}

func redactedProtocolEvents(events []protocol.Event) []protocol.Event {
	redacted := make([]protocol.Event, 0, len(events))
	for _, event := range events {
		event.ToolCall = nil
		redacted = append(redacted, event)
	}
	return redacted
}

func buildPolicyDecision(projectName, transformID, transformRunID, artifactVersionID, outputName string, policyResource *manifest.PolicyResource, decision policy.Decision) state.PolicyDecision {
	policyID := ""
	if policyResource != nil {
		policyID = policyResource.UniqueID
	}
	checks := make([]state.PolicyCheck, 0, len(decision.Checks))
	for _, check := range decision.Checks {
		checks = append(checks, state.PolicyCheck{
			Name:    check.Name,
			Status:  check.Status,
			Message: check.Message,
		})
	}
	return state.PolicyDecision{
		DecisionID:        newID("policy_decision." + projectName + "." + outputName),
		PolicyID:          policyID,
		TransformID:       transformID,
		TransformRunID:    transformRunID,
		ArtifactVersionID: artifactVersionID,
		Status:            decision.Status,
		Checks:            checks,
		DecidedAt:         time.Now().UTC().Format(time.RFC3339),
	}
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

func protocolInputs(root string, transform manifest.TransformResource, m manifest.Manifest, snapshot state.Snapshot, versions state.ArtifactVersionsIndex) []any {
	inputs := make([]any, 0, len(transform.Inputs))
	for _, input := range transform.Inputs {
		record := map[string]any{
			"kind":      input.Kind,
			"name":      input.Name,
			"unique_id": input.UniqueID,
		}
		if len(input.Require) > 0 {
			record["require"] = input.Require
		}
		switch input.Kind {
		case "source":
			if source, ok := m.Sources[input.UniqueID]; ok {
				record["resource_type"] = source.ResourceType
				record["source_name"] = source.SourceName
				record["artifact_type"] = source.ArtifactType
				record["path"] = source.Path
				record["resolved_paths"] = source.ResolvedPaths
				record["fingerprint"] = source.Fingerprint
				record["tags"] = source.Tags
				if len(source.Meta) > 0 {
					record["meta"] = source.Meta
				}
				if descriptor, semantic, ok := sourceDescriptors(root, source); ok {
					record["descriptor"] = descriptor
					if len(semantic) > 0 {
						record["semantic_descriptor"] = semantic
					}
				}
			}
		case "ref":
			if artifactResource, ok := m.Artifacts[input.UniqueID]; ok {
				record["resource_type"] = artifactResource.ResourceType
				record["artifact_type"] = artifactResource.ArtifactType
				record["logical_path"] = artifactResource.LogicalPath
				if len(artifactResource.Contract) > 0 {
					record["contract"] = artifactResource.Contract
				}
				if len(artifactResource.Tags) > 0 {
					record["tags"] = artifactResource.Tags
				}
			}
			if pointer, ok := snapshot.CurrentArtifacts[input.UniqueID]; ok {
				record["current"] = pointer
				if version, ok := versions.ArtifactVersions[pointer.CurrentVersionID]; ok {
					record["current_version"] = map[string]any{
						"version_id":          version.VersionID,
						"artifact_id":         version.ArtifactID,
						"logical_path":        version.LogicalPath,
						"storage_path":        version.StoragePath,
						"absolute_path":       filepath.Join(root, version.StoragePath),
						"descriptor":          version.Descriptor,
						"semantic_descriptor": version.SemanticDescriptor,
						"confidence":          version.Confidence,
						"generated_by":        version.GeneratedBy,
					}
				}
			}
		}
		inputs = append(inputs, record)
	}
	return inputs
}

func sourceDescriptors(root string, source manifest.SourceResource) (artifact.Descriptor, map[string]any, bool) {
	if strings.ContainsAny(source.Path, "*?[") || source.Path == "" {
		return artifact.Descriptor{}, nil, false
	}
	descriptor, err := artifact.Describe(root, source.Path, source.ArtifactType)
	if err != nil {
		return artifact.Descriptor{}, nil, false
	}
	semantic, err := artifact.SemanticDescriptor(root, source.Path, source.ArtifactType)
	if err != nil {
		semantic = nil
	}
	return descriptor, semantic, true
}

func protocolAssets(root string, transform manifest.TransformResource, m manifest.Manifest) []any {
	assets := make([]any, 0, len(transform.Assets))
	for _, assetID := range transform.Assets {
		asset, ok := m.TransformAssets[assetID]
		if !ok {
			continue
		}
		record := map[string]any{
			"unique_id":     asset.UniqueID,
			"resource_type": asset.ResourceType,
			"name":          asset.Name,
			"asset_type":    asset.AssetType,
			"path":          asset.Path,
			"absolute_path": filepath.Join(root, asset.Path),
			"fingerprint":   asset.Fingerprint,
			"variables":     asset.Variables,
		}
		if len(asset.Meta) > 0 {
			record["meta"] = asset.Meta
		}
		assets = append(assets, record)
	}
	return assets
}

func protocolRunner(transform manifest.TransformResource, m manifest.Manifest, resolved runner.Resolved) map[string]any {
	record := map[string]any{
		"unique_id": transform.Runner,
		"name":      resolved.Name,
		"type":      resolved.Type,
		"protocol":  resolved.Protocol,
		"source":    resolved.Source,
	}
	if runnerResource, ok := m.Runners[transform.Runner]; ok {
		record["unique_id"] = runnerResource.UniqueID
		record["name"] = runnerResource.Name
		record["type"] = runnerResource.RunnerType
		record["protocol"] = runnerResource.Protocol
		record["env"] = runnerResource.Env
		record["config"] = runnerResource.Config
		record["capabilities"] = runnerResource.Capabilities
		record["fingerprint"] = runnerResource.Fingerprint
		record["args"] = runnerResource.Args
		record["cwd"] = runnerResource.CWD
	}
	if resolved.PluginName != "" {
		record["plugin_name"] = resolved.PluginName
	}
	if resolved.Version != "" {
		record["version"] = resolved.Version
	}
	return record
}

func protocolTools(transform manifest.TransformResource) []any {
	tools := make([]any, 0, len(transform.Tools))
	for _, tool := range transform.Tools {
		tools = append(tools, map[string]any{"name": tool})
	}
	return tools
}

func protocolState(transformID string, transform manifest.TransformResource, snapshot state.Snapshot, versions state.ArtifactVersionsIndex, node planner.Node) map[string]any {
	currentOutputs := map[string]any{}
	for _, output := range transform.Outputs {
		pointer, ok := snapshot.CurrentArtifacts[output.UniqueID]
		if !ok {
			continue
		}
		record := map[string]any{"pointer": pointer}
		if version, ok := versions.ArtifactVersions[pointer.CurrentVersionID]; ok {
			record["version"] = map[string]any{
				"version_id":          version.VersionID,
				"storage_path":        version.StoragePath,
				"descriptor":          version.Descriptor,
				"semantic_descriptor": version.SemanticDescriptor,
				"confidence":          version.Confidence,
				"generated_by":        version.GeneratedBy,
			}
		}
		currentOutputs[output.UniqueID] = record
	}
	statePayload := map[string]any{
		"transform_id":    transformID,
		"plan":            map[string]any{"action": node.Action, "dirty_reasons": node.DirtyReasons, "blocked_reasons": node.BlockedReasons},
		"current_outputs": currentOutputs,
	}
	if latest, ok := snapshot.LatestRuns[transformID]; ok {
		statePayload["previous_run"] = latest
	}
	return statePayload
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
	return graph.SelectTransforms(m, expr)
}

func maxConfidence(left, right string) string {
	order := map[string]int{
		"experimental": 0,
		"structural":   1,
		"semantic":     2,
		"exact":        3,
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
