package graph

import (
	"fmt"
	"sort"
	"strings"

	"github.com/nyuta01/fbt/internal/config"
	"github.com/nyuta01/fbt/internal/manifest"
	"github.com/nyuta01/fbt/internal/state"
)

type Selector struct {
	Method string
	Value  string
}

func SelectTransforms(m manifest.Manifest, expr string) (map[string]struct{}, error) {
	return SelectTransformsWithState(m, state.Snapshot{}, expr)
}

func SelectTransformsWithState(m manifest.Manifest, snapshot state.Snapshot, expr string) (map[string]struct{}, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, nil
	}
	selection, err := parseTransformSelection(expr)
	if err != nil {
		return nil, err
	}
	ids, err := selectBaseWithState(m, snapshot, selection.Base)
	if err != nil {
		return nil, err
	}
	selected := map[string]struct{}{}
	for _, id := range ids {
		if _, ok := m.Transforms[id]; ok {
			selected[id] = struct{}{}
		}
	}
	if selection.NameSelector && len(selected) > 1 {
		return nil, fmt.Errorf("ambiguous selector %q matched multiple transforms: %s", expr, strings.Join(sortedIDs(selected), ", "))
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("selector matched no transforms: %s", expr)
	}
	seeds := sortedIDs(selected)
	if selection.Upstream {
		addRelatedTransforms(m, seeds, m.ParentMap, selected)
	}
	if selection.Downstream {
		addRelatedTransforms(m, seeds, m.ChildMap, selected)
	}
	return selected, nil
}

type transformSelection struct {
	Base         string
	Upstream     bool
	Downstream   bool
	NameSelector bool
}

func parseTransformSelection(expr string) (transformSelection, error) {
	selection := transformSelection{
		Base:       expr,
		Upstream:   strings.HasPrefix(expr, "+"),
		Downstream: strings.HasSuffix(expr, "+"),
	}
	if selection.Upstream {
		selection.Base = strings.TrimPrefix(selection.Base, "+")
	}
	if selection.Downstream {
		selection.Base = strings.TrimSuffix(selection.Base, "+")
	}
	if selection.Base == "" || strings.HasPrefix(selection.Base, "+") || strings.HasSuffix(selection.Base, "+") {
		return transformSelection{}, fmt.Errorf("invalid graph selector %q: use +target, target+, or +target+", expr)
	}
	selection.NameSelector = !strings.Contains(selection.Base, ":")
	return selection, nil
}

func selectBase(m manifest.Manifest, expr string) ([]string, error) {
	return selectBaseWithState(m, state.Snapshot{}, expr)
}

func selectBaseWithState(m manifest.Manifest, snapshot state.Snapshot, expr string) ([]string, error) {
	switch {
	case strings.HasPrefix(expr, "selector:"):
		name := strings.TrimPrefix(expr, "selector:")
		definition, ok := m.Selectors[name]
		if !ok {
			return nil, fmt.Errorf("unknown selector: %s", name)
		}
		return SelectDefinitionWithState(m, snapshot, definition)
	case strings.HasPrefix(expr, "tag:"):
		return SelectWithState(m, snapshot, Selector{Method: "tag", Value: strings.TrimPrefix(expr, "tag:")})
	case strings.HasPrefix(expr, "path:"):
		return SelectWithState(m, snapshot, Selector{Method: "path", Value: strings.TrimPrefix(expr, "path:")})
	case strings.HasPrefix(expr, "resource_type:"):
		return SelectWithState(m, snapshot, Selector{Method: "resource_type", Value: strings.TrimPrefix(expr, "resource_type:")})
	case strings.HasPrefix(expr, "state:"):
		return SelectWithState(m, snapshot, Selector{Method: "state", Value: strings.TrimPrefix(expr, "state:")})
	default:
		return SelectWithState(m, snapshot, Selector{Method: "name", Value: expr})
	}
}

func addRelatedTransforms(m manifest.Manifest, seeds []string, relationMap map[string][]string, selected map[string]struct{}) {
	seen := map[string]struct{}{}
	queue := append([]string(nil), seeds...)
	for _, seed := range seeds {
		seen[seed] = struct{}{}
	}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		for _, related := range relationMap[id] {
			if _, ok := seen[related]; ok {
				continue
			}
			seen[related] = struct{}{}
			if _, ok := m.Transforms[related]; ok {
				selected[related] = struct{}{}
			}
			queue = append(queue, related)
		}
	}
}

func Select(m manifest.Manifest, selector Selector) ([]string, error) {
	return SelectWithState(m, state.Snapshot{}, selector)
}

func SelectWithState(m manifest.Manifest, snapshot state.Snapshot, selector Selector) ([]string, error) {
	switch selector.Method {
	case "name":
		return selectByName(m, selector.Value), nil
	case "tag":
		return selectByTag(m, selector.Value), nil
	case "path":
		return selectByPath(m, selector.Value), nil
	case "resource_type":
		return selectByResourceType(m, selector.Value), nil
	case "parent":
		return selectRelations(m, selector.Value, m.ParentMap), nil
	case "child":
		return selectRelations(m, selector.Value, m.ChildMap), nil
	case "state":
		return selectByState(m, snapshot, selector.Value), nil
	default:
		return nil, fmt.Errorf("unsupported selector method %q", selector.Method)
	}
}

func SelectDefinition(m manifest.Manifest, definition config.SelectorDefinition) ([]string, error) {
	return SelectDefinitionWithState(m, state.Snapshot{}, definition)
}

func SelectDefinitionWithState(m manifest.Manifest, snapshot state.Snapshot, definition config.SelectorDefinition) ([]string, error) {
	if len(definition.Union) > 0 {
		seen := map[string]struct{}{}
		for _, child := range definition.Union {
			selected, err := SelectDefinitionWithState(m, snapshot, child)
			if err != nil {
				return nil, err
			}
			for _, id := range selected {
				seen[id] = struct{}{}
			}
		}
		return sortedIDs(seen), nil
	}
	return SelectWithState(m, snapshot, Selector{Method: definition.Method, Value: definition.Value})
}

func selectByName(m manifest.Manifest, value string) []string {
	seen := map[string]struct{}{}
	for id, summary := range m.ResourceSummaries() {
		if summary.Name == value || summary.UniqueID == value || strings.HasSuffix(summary.UniqueID, "."+value) {
			seen[id] = struct{}{}
		}
	}
	return sortedIDs(seen)
}

func selectByTag(m manifest.Manifest, value string) []string {
	seen := map[string]struct{}{}
	for id, summary := range m.ResourceSummaries() {
		for _, tag := range summary.Tags {
			if tag == value {
				seen[id] = struct{}{}
			}
		}
	}
	return sortedIDs(seen)
}

func selectByPath(m manifest.Manifest, value string) []string {
	needle := strings.TrimPrefix(strings.ReplaceAll(value, "\\", "/"), "./")
	seen := map[string]struct{}{}
	for id, summary := range m.ResourceSummaries() {
		path := strings.TrimPrefix(strings.ReplaceAll(summary.Path, "\\", "/"), "./")
		if path == needle || strings.HasPrefix(path, strings.TrimSuffix(needle, "/")+"/") || strings.HasPrefix(needle, strings.TrimSuffix(path, "/")+"/") {
			seen[id] = struct{}{}
		}
	}
	return sortedIDs(seen)
}

func selectByResourceType(m manifest.Manifest, value string) []string {
	seen := map[string]struct{}{}
	for id, summary := range m.ResourceSummaries() {
		if summary.ResourceType == value {
			seen[id] = struct{}{}
		}
	}
	return sortedIDs(seen)
}

func selectByState(m manifest.Manifest, snapshot state.Snapshot, value string) []string {
	seen := map[string]struct{}{}
	for id := range m.Transforms {
		latest, ok := snapshot.LatestRuns[id]
		if !ok {
			continue
		}
		if value == "failed" {
			if state.IsFailedLatestStatus(latest.LatestStatus) {
				seen[id] = struct{}{}
			}
			continue
		}
		if latest.LatestStatus == value {
			seen[id] = struct{}{}
		}
	}
	return sortedIDs(seen)
}

func selectRelations(m manifest.Manifest, value string, relationMap map[string][]string) []string {
	seen := map[string]struct{}{}
	for _, id := range selectByName(m, value) {
		for _, related := range relationMap[id] {
			seen[related] = struct{}{}
		}
	}
	return sortedIDs(seen)
}

func sortedIDs(seen map[string]struct{}) []string {
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
