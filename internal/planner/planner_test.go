package planner

import (
	"slices"
	"testing"

	"github.com/nyuta01/fbt/internal/manifest"
	"github.com/nyuta01/fbt/internal/state"
)

func TestBuildPlansRunForMissingOutput(t *testing.T) {
	m := fixtureManifest("asset-old")
	plan := Build(Inputs{Manifest: m, State: emptyState()})

	node := plan.Nodes[0]
	if node.Action != ActionRun {
		t.Fatalf("expected run, got %s", node.Action)
	}
	assertContains(t, node.DirtyReasons, "output missing")
	assertContains(t, node.DirtyReasons, "no previous successful run")
}

func TestBuildSkipsCleanTransform(t *testing.T) {
	m := fixtureManifest("asset-old")
	transformID := manifest.TransformID("knowledge_ops", "case_summaries")
	artifactID := manifest.ArtifactID("knowledge_ops", "case_summaries")
	snapshot := emptyState()
	snapshot.CurrentArtifacts[artifactID] = state.ArtifactPointer{
		ArtifactID:       artifactID,
		CurrentVersionID: "artifact_version.knowledge_ops.case_summaries.sha256_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		CurrentDigest:    "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		LogicalPath:      "target/artifacts/support/case_summaries/",
	}
	snapshot.LatestRuns[transformID] = state.LatestRun{
		LatestRunID:                "transform_run.run_1",
		LatestSuccessfulRunID:      "transform_run.run_1",
		LatestStatus:               "success",
		LatestEffectiveFingerprint: m.Transforms[transformID].Fingerprint["effective"],
	}

	plan := Build(Inputs{Manifest: m, State: snapshot})
	if plan.Nodes[0].Action != ActionSkip {
		t.Fatalf("expected skip, got %+v", plan.Nodes[0])
	}
}

func TestBuildDetectsManifestDirtyReasons(t *testing.T) {
	previous := fixtureManifest("asset-old")
	current := fixtureManifest("asset-new")
	transformID := manifest.TransformID("knowledge_ops", "case_summaries")
	artifactID := manifest.ArtifactID("knowledge_ops", "case_summaries")
	snapshot := emptyState()
	snapshot.CurrentArtifacts[artifactID] = state.ArtifactPointer{ArtifactID: artifactID}
	snapshot.LatestRuns[transformID] = state.LatestRun{
		LatestRunID:                "transform_run.run_1",
		LatestSuccessfulRunID:      "transform_run.run_1",
		LatestStatus:               "success",
		LatestEffectiveFingerprint: previous.Transforms[transformID].Fingerprint["effective"],
	}

	plan := Build(Inputs{Manifest: current, PreviousManifest: &previous, State: snapshot})
	node := plan.Nodes[0]
	if node.Action != ActionRun {
		t.Fatalf("expected run, got %+v", node)
	}
	assertContains(t, node.DirtyReasons, "effective fingerprint changed")
	assertContains(t, node.DirtyReasons, "transform asset changed")
}

func TestBuildBlocksOnReviewAndConfidenceRequirements(t *testing.T) {
	m := fixtureManifest("asset-old")
	weeklyID := manifest.TransformID("knowledge_ops", "weekly_support_insights")
	caseID := manifest.ArtifactID("knowledge_ops", "case_summaries")
	m.Transforms[weeklyID] = manifest.TransformResource{
		UniqueID:      weeklyID,
		ResourceType:  "transform",
		Name:          "weekly_support_insights",
		TransformType: "agent",
		Inputs: []manifest.TransformInput{
			{
				Kind:     "ref",
				UniqueID: caseID,
				Name:     "case_summaries",
				Require: map[string]any{
					"confidence": "reviewed",
					"review": map[string]any{
						"status": "approved",
					},
				},
			},
		},
		Outputs: []manifest.TransformOutput{
			{UniqueID: manifest.ArtifactID("knowledge_ops", "weekly_support_insights"), Name: "weekly_support_insights"},
		},
		Fingerprint: map[string]string{"effective": "weekly"},
	}
	m.ParentMap[weeklyID] = []string{caseID}

	snapshot := emptyState()
	snapshot.CurrentArtifacts[caseID] = state.ArtifactPointer{
		ArtifactID:     caseID,
		Confidence:     "semantic",
		ApprovalStatus: "pending",
	}

	plan := Build(Inputs{Manifest: m, State: snapshot, Selected: map[string]struct{}{weeklyID: {}}})
	node := plan.Nodes[0]
	if node.Action != ActionBlocked {
		t.Fatalf("expected blocked, got %+v", node)
	}
	assertContains(t, node.BlockedReasons, "requires artifact.knowledge_ops.case_summaries confidence reviewed, current is semantic")
	assertContains(t, node.BlockedReasons, "requires artifact.knowledge_ops.case_summaries review status approved, current is pending")
}

func TestBuildAllowsApprovedReviewedInput(t *testing.T) {
	m := fixtureManifest("asset-old")
	weeklyID := manifest.TransformID("knowledge_ops", "weekly_support_insights")
	caseID := manifest.ArtifactID("knowledge_ops", "case_summaries")
	m.Transforms[weeklyID] = manifest.TransformResource{
		UniqueID:      weeklyID,
		ResourceType:  "transform",
		Name:          "weekly_support_insights",
		TransformType: "agent",
		Inputs: []manifest.TransformInput{
			{
				Kind:     "ref",
				UniqueID: caseID,
				Name:     "case_summaries",
				Require: map[string]any{
					"confidence": "reviewed",
					"review": map[string]any{
						"status": "approved",
					},
				},
			},
		},
		Outputs: []manifest.TransformOutput{
			{UniqueID: manifest.ArtifactID("knowledge_ops", "weekly_support_insights"), Name: "weekly_support_insights"},
		},
		Fingerprint: map[string]string{"effective": "weekly"},
	}
	m.ParentMap[weeklyID] = []string{caseID}

	snapshot := emptyState()
	snapshot.CurrentArtifacts[caseID] = state.ArtifactPointer{
		ArtifactID:     caseID,
		Confidence:     "reviewed",
		ApprovalStatus: "approved",
	}

	plan := Build(Inputs{Manifest: m, State: snapshot, Selected: map[string]struct{}{weeklyID: {}}})
	node := plan.Nodes[0]
	if node.Action == ActionBlocked {
		t.Fatalf("expected approved input not to block, got %+v", node)
	}
}

func TestBuildHonorsSelectedSet(t *testing.T) {
	m := fixtureManifest("asset-old")
	selected := map[string]struct{}{
		manifest.TransformID("knowledge_ops", "case_summaries"): {},
	}
	plan := Build(Inputs{Manifest: m, State: emptyState(), Selected: selected})
	if len(plan.Nodes) != 1 {
		t.Fatalf("expected one selected node, got %d", len(plan.Nodes))
	}
}

func fixtureManifest(assetFingerprint string) manifest.Manifest {
	project := "knowledge_ops"
	sourceID := manifest.SourceID(project, "support", "raw_tickets")
	assetID := manifest.TransformAssetID(project, "case_summary_prompt")
	policyID := manifest.PolicyID(project, "support_agent_scope")
	evalID := manifest.EvalID(project, "required_sections")
	runnerID := manifest.RunnerID(project, "openai.responses")
	transformID := manifest.TransformID(project, "case_summaries")
	artifactID := manifest.ArtifactID(project, "case_summaries")
	return manifest.Manifest{
		Sources: map[string]manifest.SourceResource{
			sourceID: {UniqueID: sourceID, ResourceType: "source", Fingerprint: manifest.Fingerprint{Value: "source"}},
		},
		Artifacts: map[string]manifest.ArtifactResource{
			artifactID: {UniqueID: artifactID, ResourceType: "artifact", Name: "case_summaries"},
		},
		Transforms: map[string]manifest.TransformResource{
			transformID: {
				UniqueID:      transformID,
				ResourceType:  "transform",
				Name:          "case_summaries",
				TransformType: "llm",
				Inputs: []manifest.TransformInput{
					{Kind: "source", UniqueID: sourceID, Name: "support.raw_tickets"},
				},
				Outputs: []manifest.TransformOutput{
					{UniqueID: artifactID, Name: "case_summaries"},
				},
				Model: map[string]any{"name": "gpt-5"},
				Fingerprint: map[string]string{
					"config":    "config",
					"effective": "effective-" + assetFingerprint,
				},
			},
		},
		TransformAssets: map[string]manifest.TransformAssetResource{
			assetID: {UniqueID: assetID, ResourceType: "transform_asset", Fingerprint: manifest.Fingerprint{Value: assetFingerprint}},
		},
		Policies: map[string]manifest.PolicyResource{
			policyID: {UniqueID: policyID, ResourceType: "policy", Fingerprint: manifest.Fingerprint{Value: "policy"}},
		},
		Evals: map[string]manifest.EvalResource{
			evalID: {UniqueID: evalID, ResourceType: "eval", Fingerprint: manifest.Fingerprint{Value: "eval"}},
		},
		Runners: map[string]manifest.RunnerResource{
			runnerID: {UniqueID: runnerID, ResourceType: "runner", Fingerprint: manifest.Fingerprint{Value: "runner"}},
		},
		ParentMap: map[string][]string{
			transformID: {sourceID, assetID, policyID, evalID, runnerID},
		},
		ChildMap: map[string][]string{},
	}
}

func emptyState() state.Snapshot {
	return state.Snapshot{
		CurrentArtifacts: map[string]state.ArtifactPointer{},
		LatestRuns:       map[string]state.LatestRun{},
	}
}

func assertContains(t *testing.T, values []string, want string) {
	t.Helper()
	if !slices.Contains(values, want) {
		t.Fatalf("expected %q in %v", want, values)
	}
}
