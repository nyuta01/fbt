package planner

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/nyuta01/fbt/internal/manifest"
	"github.com/nyuta01/fbt/internal/state"
)

type Action string

const (
	ActionRun     Action = "run"
	ActionSkip    Action = "skip"
	ActionBlocked Action = "blocked"
)

type Inputs struct {
	Manifest         manifest.Manifest
	PreviousManifest *manifest.Manifest
	State            state.Snapshot
	Selected         map[string]struct{}
}

type Plan struct {
	Nodes   []Node  `json:"nodes"`
	Summary Summary `json:"summary"`
}

type Summary struct {
	Selected int `json:"selected"`
	Run      int `json:"run"`
	Skipped  int `json:"skipped"`
	Blocked  int `json:"blocked"`
}

type Node struct {
	TransformID    string   `json:"transform_id"`
	Name           string   `json:"name"`
	Action         Action   `json:"action"`
	DirtyReasons   []string `json:"dirty_reasons,omitempty"`
	BlockedReasons []string `json:"blocked_reasons,omitempty"`
	Outputs        []string `json:"outputs,omitempty"`
}

func Build(inputs Inputs) Plan {
	ids := make([]string, 0, len(inputs.Manifest.Transforms))
	for id := range inputs.Manifest.Transforms {
		if len(inputs.Selected) > 0 {
			if _, ok := inputs.Selected[id]; !ok {
				continue
			}
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)

	plan := Plan{}
	for _, id := range ids {
		transform := inputs.Manifest.Transforms[id]
		node := Node{
			TransformID: id,
			Name:        transform.Name,
			Outputs:     outputIDs(transform),
		}
		node.BlockedReasons = blockedReasons(transform, inputs.State)
		if len(node.BlockedReasons) > 0 {
			node.Action = ActionBlocked
			plan.Summary.Blocked++
		} else {
			node.DirtyReasons = dirtyReasons(id, transform, inputs)
			if len(node.DirtyReasons) > 0 {
				node.Action = ActionRun
				plan.Summary.Run++
			} else {
				node.Action = ActionSkip
				plan.Summary.Skipped++
			}
		}
		plan.Nodes = append(plan.Nodes, node)
		plan.Summary.Selected++
	}
	return plan
}

func blockedReasons(transform manifest.TransformResource, snapshot state.Snapshot) []string {
	var reasons []string
	for _, input := range transform.Inputs {
		if input.Kind != "ref" {
			continue
		}
		pointer, ok := snapshot.CurrentArtifacts[input.UniqueID]
		if !ok {
			reasons = append(reasons, fmt.Sprintf("requires %s current artifact", input.UniqueID))
			continue
		}
		if required, ok := stringFromMap(input.Require, "confidence"); ok && !confidenceSatisfies(pointer.Confidence, required) {
			reasons = append(reasons, fmt.Sprintf("requires %s confidence %s, current is %s", input.UniqueID, required, emptyAsUnknown(pointer.Confidence)))
		}
		if review, ok := mapFromMap(input.Require, "review"); ok {
			if requiredStatus, ok := stringFromMap(review, "status"); ok && pointer.ApprovalStatus != requiredStatus {
				reasons = append(reasons, fmt.Sprintf("requires %s review status %s, current is %s", input.UniqueID, requiredStatus, emptyAsUnknown(pointer.ApprovalStatus)))
			}
		}
	}
	sort.Strings(reasons)
	return reasons
}

func dirtyReasons(id string, transform manifest.TransformResource, inputs Inputs) []string {
	reasons := map[string]struct{}{}
	for _, output := range transform.Outputs {
		if _, ok := inputs.State.CurrentArtifacts[output.UniqueID]; !ok {
			reasons["output missing"] = struct{}{}
		}
	}

	latest, ok := inputs.State.LatestRuns[id]
	if !ok {
		reasons["no previous successful run"] = struct{}{}
	} else if latest.LatestEffectiveFingerprint != "" && latest.LatestEffectiveFingerprint != transform.Fingerprint["effective"] {
		reasons["effective fingerprint changed"] = struct{}{}
	}

	if inputs.PreviousManifest != nil {
		addManifestDiffReasons(reasons, id, transform, inputs.Manifest, *inputs.PreviousManifest)
	}

	return sortedReasons(reasons)
}

func addManifestDiffReasons(reasons map[string]struct{}, id string, transform manifest.TransformResource, current manifest.Manifest, previous manifest.Manifest) {
	previousTransform, ok := previous.Transforms[id]
	if !ok {
		reasons["transform added"] = struct{}{}
		return
	}
	if previousTransform.Fingerprint["config"] != transform.Fingerprint["config"] {
		reasons["transform config changed"] = struct{}{}
	}
	if hashAny(previousTransform.Model) != hashAny(transform.Model) {
		reasons["model parameters changed"] = struct{}{}
	}

	for _, parent := range current.ParentMap[id] {
		switch {
		case hasPrefix(parent, "source."):
			if previousSource, ok := previous.Sources[parent]; !ok || previousSource.Fingerprint != current.Sources[parent].Fingerprint {
				reasons["source descriptor changed"] = struct{}{}
			}
		case hasPrefix(parent, "transform_asset."):
			if previousAsset, ok := previous.TransformAssets[parent]; !ok || previousAsset.Fingerprint != current.TransformAssets[parent].Fingerprint {
				reasons["transform asset changed"] = struct{}{}
			}
		case hasPrefix(parent, "policy."):
			if previousPolicy, ok := previous.Policies[parent]; !ok || previousPolicy.Fingerprint != current.Policies[parent].Fingerprint {
				reasons["policy changed"] = struct{}{}
			}
		case hasPrefix(parent, "eval."):
			if previousEval, ok := previous.Evals[parent]; !ok || previousEval.Fingerprint != current.Evals[parent].Fingerprint {
				reasons["eval changed"] = struct{}{}
			}
		case hasPrefix(parent, "runner."):
			if previousRunner, ok := previous.Runners[parent]; !ok || previousRunner.Fingerprint != current.Runners[parent].Fingerprint {
				reasons["runner identity changed"] = struct{}{}
			}
		}
	}
}

func outputIDs(transform manifest.TransformResource) []string {
	outputs := make([]string, 0, len(transform.Outputs))
	for _, output := range transform.Outputs {
		outputs = append(outputs, output.UniqueID)
	}
	sort.Strings(outputs)
	return outputs
}

func confidenceSatisfies(current, required string) bool {
	if current == required {
		return true
	}
	order := map[string]int{
		"experimental": 0,
		"structural":   1,
		"semantic":     2,
		"exact":        3,
		"reviewed":     4,
	}
	currentRank, currentOK := order[current]
	requiredRank, requiredOK := order[required]
	return currentOK && requiredOK && currentRank >= requiredRank
}

func stringFromMap(values map[string]any, key string) (string, bool) {
	value, ok := values[key]
	if !ok {
		return "", false
	}
	typed, ok := value.(string)
	return typed, ok
}

func mapFromMap(values map[string]any, key string) (map[string]any, bool) {
	value, ok := values[key]
	if !ok {
		return nil, false
	}
	typed, ok := value.(map[string]any)
	return typed, ok
}

func emptyAsUnknown(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}

func sortedReasons(reasons map[string]struct{}) []string {
	out := make([]string, 0, len(reasons))
	for reason := range reasons {
		out = append(out, reason)
	}
	sort.Strings(out)
	return out
}

func hasPrefix(value, prefix string) bool {
	return len(value) >= len(prefix) && value[:len(prefix)] == prefix
}

func hashAny(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%#v", value)
	}
	return string(data)
}
