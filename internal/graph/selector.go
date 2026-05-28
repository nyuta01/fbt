package graph

import (
	"fmt"
	"sort"
	"strings"

	"github.com/nyuta01/fbt/internal/config"
	"github.com/nyuta01/fbt/internal/manifest"
)

type Selector struct {
	Method string
	Value  string
}

func Select(m manifest.Manifest, selector Selector) ([]string, error) {
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
	default:
		return nil, fmt.Errorf("unsupported selector method %q", selector.Method)
	}
}

func SelectDefinition(m manifest.Manifest, definition config.SelectorDefinition) ([]string, error) {
	if len(definition.Union) > 0 {
		seen := map[string]struct{}{}
		for _, child := range definition.Union {
			selected, err := SelectDefinition(m, child)
			if err != nil {
				return nil, err
			}
			for _, id := range selected {
				seen[id] = struct{}{}
			}
		}
		return sortedIDs(seen), nil
	}
	return Select(m, Selector{Method: definition.Method, Value: definition.Value})
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
