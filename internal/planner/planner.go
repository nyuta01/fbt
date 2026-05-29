package planner

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

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
	Force            bool
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
	TransformID    string         `json:"transform_id"`
	Name           string         `json:"name"`
	Action         Action         `json:"action"`
	DirtyReasons   []string       `json:"dirty_reasons,omitempty"`
	SourceChanges  []SourceChange `json:"source_changes,omitempty"`
	BlockedReasons []string       `json:"blocked_reasons,omitempty"`
	NextSteps      []string       `json:"next_steps,omitempty"`
	Outputs        []string       `json:"outputs,omitempty"`
}

type SourceChange struct {
	SourceID string   `json:"source_id"`
	Name     string   `json:"name,omitempty"`
	Added    []string `json:"added,omitempty"`
	Changed  []string `json:"changed,omitempty"`
	Removed  []string `json:"removed,omitempty"`
}

func Build(inputs Inputs) Plan {
	selected := selectedTransformSet(inputs.Manifest, inputs.Selected)
	ids := dependencyOrderedTransformIDs(inputs.Manifest, selected)
	selectedProducers := selectedArtifactProducers(inputs.Manifest, selected)

	plan := Plan{}
	plannedRunOutputs := map[string]struct{}{}
	for _, id := range ids {
		transform := inputs.Manifest.Transforms[id]
		node := Node{
			TransformID: id,
			Name:        transform.Name,
			Outputs:     outputIDs(transform),
		}
		node.BlockedReasons = blockedReasons(transform, inputs.State, selectedProducers)
		if len(node.BlockedReasons) > 0 {
			node.Action = ActionBlocked
			plan.Summary.Blocked++
		} else {
			node.DirtyReasons, node.SourceChanges = dirtyReasons(id, transform, inputs, plannedRunOutputs)
			if len(node.DirtyReasons) > 0 {
				node.Action = ActionRun
				plan.Summary.Run++
				for _, output := range transform.Outputs {
					plannedRunOutputs[output.UniqueID] = struct{}{}
				}
			} else {
				node.Action = ActionSkip
				plan.Summary.Skipped++
			}
		}
		node.NextSteps = nextSteps(transform, node.Action, inputs.State)
		plan.Nodes = append(plan.Nodes, node)
		plan.Summary.Selected++
	}
	return plan
}

func RuntimeBlock(transform manifest.TransformResource, snapshot state.Snapshot) ([]string, []string) {
	reasons := blockedReasons(transform, snapshot, nil)
	if len(reasons) == 0 {
		return nil, nil
	}
	return reasons, blockedNextSteps(transform, snapshot)
}

func nextSteps(transform manifest.TransformResource, action Action, snapshot state.Snapshot) []string {
	switch action {
	case ActionRun:
		return runNextSteps(transform)
	case ActionBlocked:
		return blockedNextSteps(transform, snapshot)
	case ActionSkip:
		return skippedNextSteps(transform)
	default:
		return nil
	}
}

func runNextSteps(transform manifest.TransformResource) []string {
	target := targetName(transform.Name, transform.UniqueID)
	if target == "" {
		return nil
	}
	return []string{fmt.Sprintf("fbt build --select %s", target)}
}

func blockedNextSteps(transform manifest.TransformResource, snapshot state.Snapshot) []string {
	var steps []string
	for _, input := range transform.Inputs {
		if input.Kind != "ref" {
			continue
		}
		target := targetName(input.Name, input.UniqueID)
		pointer, ok := snapshot.CurrentArtifacts[input.UniqueID]
		if !ok {
			steps = append(steps, fmt.Sprintf("fbt build --select %s", target))
			continue
		}
		if required, ok := stringFromMap(input.Require, "confidence"); ok && !confidenceSatisfies(pointer.Confidence, required) {
			steps = append(steps, fmt.Sprintf("fbt artifact explain %s", target))
		}
	}
	return uniqueStrings(steps)
}

func skippedNextSteps(transform manifest.TransformResource) []string {
	var steps []string
	for _, output := range transform.Outputs {
		target := targetName(output.Name, output.UniqueID)
		steps = append(steps, fmt.Sprintf("fbt artifact show %s", target))
	}
	if len(steps) == 0 && transform.Name != "" {
		steps = append(steps, fmt.Sprintf("fbt plan --select %s", transform.Name))
	}
	return uniqueStrings(steps)
}

func blockedReasons(transform manifest.TransformResource, snapshot state.Snapshot, selectedProducers map[string]string) []string {
	var reasons []string
	for _, input := range transform.Inputs {
		if input.Kind != "ref" {
			continue
		}
		pointer, ok := snapshot.CurrentArtifacts[input.UniqueID]
		if !ok {
			if producer, ok := selectedProducers[input.UniqueID]; ok && producer != transform.UniqueID {
				continue
			}
			reasons = append(reasons, fmt.Sprintf("requires %s current artifact", input.UniqueID))
			continue
		}
		if required, ok := stringFromMap(input.Require, "confidence"); ok && !confidenceSatisfies(pointer.Confidence, required) {
			reasons = append(reasons, fmt.Sprintf("requires %s confidence %s, current is %s", input.UniqueID, required, emptyAsUnknown(pointer.Confidence)))
		}
	}
	sort.Strings(reasons)
	return reasons
}

func dirtyReasons(id string, transform manifest.TransformResource, inputs Inputs, plannedRunOutputs map[string]struct{}) ([]string, []SourceChange) {
	reasons := map[string]struct{}{}
	var sourceChanges []SourceChange
	if inputs.Force {
		reasons["forced rebuild"] = struct{}{}
	}
	for _, input := range transform.Inputs {
		if input.Kind != "ref" {
			continue
		}
		if _, ok := plannedRunOutputs[input.UniqueID]; ok {
			reasons["upstream artifact selected to run"] = struct{}{}
		}
	}
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
		addManifestDiffReasons(reasons, &sourceChanges, id, transform, inputs.Manifest, *inputs.PreviousManifest)
	}

	return sortedReasons(reasons), sourceChanges
}

func selectedTransformSet(m manifest.Manifest, selected map[string]struct{}) map[string]struct{} {
	out := map[string]struct{}{}
	for id := range m.Transforms {
		if len(selected) > 0 {
			if _, ok := selected[id]; !ok {
				continue
			}
		}
		out[id] = struct{}{}
	}
	return out
}

func selectedArtifactProducers(m manifest.Manifest, selected map[string]struct{}) map[string]string {
	producers := map[string]string{}
	for id := range selected {
		transform := m.Transforms[id]
		for _, output := range transform.Outputs {
			producers[output.UniqueID] = id
		}
	}
	return producers
}

func dependencyOrderedTransformIDs(m manifest.Manifest, selected map[string]struct{}) []string {
	ids := make([]string, 0, len(selected))
	for id := range selected {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	producers := selectedArtifactProducers(m, selected)
	children := map[string][]string{}
	indegree := map[string]int{}
	for _, id := range ids {
		indegree[id] = 0
	}
	for _, id := range ids {
		transform := m.Transforms[id]
		for _, input := range transform.Inputs {
			if input.Kind != "ref" {
				continue
			}
			producer, ok := producers[input.UniqueID]
			if !ok || producer == id {
				continue
			}
			children[producer] = append(children[producer], id)
			indegree[id]++
		}
	}
	for id := range children {
		sort.Strings(children[id])
	}

	var queue []string
	for _, id := range ids {
		if indegree[id] == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue)

	var ordered []string
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		ordered = append(ordered, id)
		for _, child := range children[id] {
			indegree[child]--
			if indegree[child] == 0 {
				queue = append(queue, child)
				sort.Strings(queue)
			}
		}
	}
	if len(ordered) == len(ids) {
		return ordered
	}

	seen := map[string]struct{}{}
	for _, id := range ordered {
		seen[id] = struct{}{}
	}
	for _, id := range ids {
		if _, ok := seen[id]; !ok {
			ordered = append(ordered, id)
		}
	}
	return ordered
}

func addManifestDiffReasons(reasons map[string]struct{}, sourceChanges *[]SourceChange, id string, transform manifest.TransformResource, current manifest.Manifest, previous manifest.Manifest) {
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
			currentSource, currentOK := current.Sources[parent]
			previousSource, previousOK := previous.Sources[parent]
			if !currentOK {
				continue
			}
			if !previousOK || previousSource.Fingerprint != currentSource.Fingerprint {
				reasons["source descriptor changed"] = struct{}{}
				if change, ok := sourceChange(current, previous, parent); ok {
					*sourceChanges = append(*sourceChanges, change)
				}
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

func sourceChange(current manifest.Manifest, previous manifest.Manifest, sourceID string) (SourceChange, bool) {
	currentFiles := filesForResource(current.Files, sourceID)
	previousFiles := filesForResource(previous.Files, sourceID)
	if len(currentFiles) == 0 && len(previousFiles) == 0 {
		return SourceChange{}, false
	}
	change := SourceChange{SourceID: sourceID, Name: sourceName(current.Sources[sourceID])}
	for path, checksum := range currentFiles {
		previousChecksum, ok := previousFiles[path]
		if !ok {
			change.Added = append(change.Added, path)
			continue
		}
		if previousChecksum != checksum {
			change.Changed = append(change.Changed, path)
		}
	}
	for path := range previousFiles {
		if _, ok := currentFiles[path]; !ok {
			change.Removed = append(change.Removed, path)
		}
	}
	sort.Strings(change.Added)
	sort.Strings(change.Changed)
	sort.Strings(change.Removed)
	return change, len(change.Added) > 0 || len(change.Changed) > 0 || len(change.Removed) > 0
}

func filesForResource(files map[string]manifest.FileResource, resourceID string) map[string]string {
	out := map[string]string{}
	for key, file := range files {
		if !containsString(file.ResourceIDs, resourceID) {
			continue
		}
		path := file.Path
		if path == "" {
			path = key
		}
		out[path] = file.Checksum
	}
	return out
}

func sourceName(source manifest.SourceResource) string {
	switch {
	case source.SourceName != "" && source.Name != "":
		return source.SourceName + "." + source.Name
	case source.Name != "":
		return source.Name
	default:
		return source.UniqueID
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

func targetName(name, id string) string {
	if name != "" {
		return name
	}
	if index := strings.LastIndex(id, "."); index >= 0 && index+1 < len(id) {
		return id[index+1:]
	}
	return id
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
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

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func hashAny(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%#v", value)
	}
	return string(data)
}
