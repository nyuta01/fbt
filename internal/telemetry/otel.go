package telemetry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/nyuta01/fbt/internal/manifest"
	"github.com/nyuta01/fbt/internal/state"
)

const (
	InstrumentationScopeName = "github.com/nyuta01/fbt"
)

type OTLPInput struct {
	Manifest         manifest.Manifest
	RunResults       []map[string]any
	ArtifactVersions state.ArtifactVersionsIndex
	FBTVersion       string
}

type TracesData struct {
	ResourceSpans []ResourceSpans `json:"resourceSpans"`
}

type ResourceSpans struct {
	Resource   Resource     `json:"resource"`
	ScopeSpans []ScopeSpans `json:"scopeSpans"`
}

type Resource struct {
	Attributes []KeyValue `json:"attributes,omitempty"`
}

type ScopeSpans struct {
	Scope Scope  `json:"scope"`
	Spans []Span `json:"spans"`
}

type Scope struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type Span struct {
	TraceID           string      `json:"traceId"`
	SpanID            string      `json:"spanId"`
	ParentSpanID      string      `json:"parentSpanId,omitempty"`
	Name              string      `json:"name"`
	Kind              int         `json:"kind,omitempty"`
	StartTimeUnixNano string      `json:"startTimeUnixNano,omitempty"`
	EndTimeUnixNano   string      `json:"endTimeUnixNano,omitempty"`
	Attributes        []KeyValue  `json:"attributes,omitempty"`
	Events            []SpanEvent `json:"events,omitempty"`
	Status            Status      `json:"status,omitempty"`
}

type SpanEvent struct {
	TimeUnixNano string     `json:"timeUnixNano,omitempty"`
	Name         string     `json:"name"`
	Attributes   []KeyValue `json:"attributes,omitempty"`
}

type Status struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type KeyValue struct {
	Key   string         `json:"key"`
	Value map[string]any `json:"value"`
}

type invocationGroup struct {
	id         string
	started    map[string]any
	completed  map[string]any
	transforms []map[string]any
}

func OTLPTraces(input OTLPInput) TracesData {
	invocations := groupInvocations(input.RunResults)
	spans := make([]Span, 0)
	for _, invocation := range invocations {
		spans = append(spans, invocationSpan(input, invocation))
		spans = append(spans, transformSpans(input, invocation)...)
	}
	return TracesData{
		ResourceSpans: []ResourceSpans{
			{
				Resource: Resource{Attributes: attributes(map[string]any{
					"service.name":           "fbt",
					"service.version":        input.FBTVersion,
					"fbt.project.name":       projectName(input.Manifest),
					"fbt.project.root":       ".",
					"telemetry.sdk.name":     "fbt",
					"telemetry.sdk.language": "go",
				})},
				ScopeSpans: []ScopeSpans{
					{
						Scope: Scope{
							Name:    InstrumentationScopeName,
							Version: input.FBTVersion,
						},
						Spans: spans,
					},
				},
			},
		},
	}
}

func WriteOTLPJSON(w io.Writer, traces TracesData) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(traces)
}

func groupInvocations(records []map[string]any) []*invocationGroup {
	byID := map[string]*invocationGroup{}
	var order []*invocationGroup
	ensure := func(id string) *invocationGroup {
		group, ok := byID[id]
		if ok {
			return group
		}
		group = &invocationGroup{id: id}
		byID[id] = group
		order = append(order, group)
		return group
	}
	for _, record := range records {
		invocationID := stringField(record, "invocation_id")
		if invocationID == "" {
			continue
		}
		group := ensure(invocationID)
		switch stringField(record, "record_type") {
		case "invocation_started":
			group.started = record
		case "invocation_completed":
			group.completed = record
		case "transform_run":
			group.transforms = append(group.transforms, record)
		}
	}
	return order
}

func invocationSpan(input OTLPInput, invocation *invocationGroup) Span {
	start := stringField(invocation.started, "started_at")
	end := stringField(invocation.completed, "completed_at")
	if end == "" {
		end = start
	}
	attrs := map[string]any{
		"fbt.invocation.id": invocation.id,
		"fbt.command":       stringField(invocation.started, "command"),
		"fbt.target.name":   stringField(invocation.started, "target_name"),
		"fbt.project.name":  projectName(input.Manifest),
		"fbt.run.status":    stringField(invocation.completed, "status"),
	}
	if summary, ok := invocation.completed["summary"].(map[string]any); ok {
		for key, value := range summary {
			attrs["fbt.summary."+key] = value
		}
	}
	return Span{
		TraceID:           traceID(invocation.id),
		SpanID:            spanID("invocation:" + invocation.id),
		Name:              "fbt build",
		Kind:              1,
		StartTimeUnixNano: unixNano(start),
		EndTimeUnixNano:   unixNano(end),
		Attributes:        attributes(attrs),
		Status:            status(stringField(invocation.completed, "status")),
	}
}

func transformSpans(input OTLPInput, invocation *invocationGroup) []Span {
	spans := make([]Span, 0, len(invocation.transforms))
	for _, record := range invocation.transforms {
		transformID := stringField(record, "transform_id")
		transform := input.Manifest.Transforms[transformID]
		start := stringField(record, "started_at")
		if start == "" {
			start = stringField(invocation.started, "started_at")
		}
		end := stringField(record, "completed_at")
		if end == "" {
			end = stringField(invocation.completed, "completed_at")
		}
		committedVersions := stringSlice(record["committed_versions"])
		artifactIDs := artifactIDs(input.ArtifactVersions, committedVersions)
		attrs := map[string]any{
			"fbt.invocation.id":         invocation.id,
			"fbt.transform.id":          transformID,
			"fbt.transform.name":        transform.Name,
			"fbt.transform.run_id":      stringField(record, "run_id"),
			"fbt.runner.id":             transform.Runner,
			"fbt.run.status":            stringField(record, "status"),
			"fbt.artifact.version_ids":  committedVersions,
			"fbt.artifact.ids":          artifactIDs,
			"fbt.evaluation.result_ids": stringSlice(record["evaluation_results"]),
			"fbt.policy.decision_ids":   stringSlice(record["policy_decisions"]),
			"fbt.policy.id":             transform.Policy,
			"fbt.duration_ms":           record["duration_ms"],
		}
		addModelAttributes(attrs, transform.Model)
		if usage, ok := record["usage"].(map[string]any); ok {
			for key, value := range usage {
				attrs[key] = value
			}
		}
		if provenance, ok := record["provenance"].(map[string]any); ok {
			for key, value := range provenance {
				attrs["fbt.provenance."+key] = value
			}
		}
		if errRecord, ok := record["error"].(map[string]any); ok {
			attrs["error.type"] = stringField(errRecord, "kind")
			attrs["error.message"] = stringField(errRecord, "message")
		}
		events := spanEvents(record["events"], start)
		if errEvent, ok := errorSpanEvent(record["error"], end); ok {
			events = append(events, errEvent)
		}
		spans = append(spans, Span{
			TraceID:           traceID(invocation.id),
			SpanID:            spanID("transform:" + stringField(record, "run_id")),
			ParentSpanID:      spanID("invocation:" + invocation.id),
			Name:              transformSpanName(transformID, transform),
			Kind:              1,
			StartTimeUnixNano: unixNano(start),
			EndTimeUnixNano:   unixNano(end),
			Attributes:        attributes(attrs),
			Events:            events,
			Status:            status(stringField(record, "status")),
		})
	}
	return spans
}

func addModelAttributes(attrs map[string]any, model map[string]any) {
	if len(model) == 0 {
		return
	}
	if provider, ok := model["provider"]; ok {
		attrs["gen_ai.provider.name"] = provider
		attrs["fbt.model.provider"] = provider
	}
	if name, ok := model["name"]; ok {
		attrs["gen_ai.request.model"] = name
		attrs["fbt.model.name"] = name
	}
}

func artifactIDs(index state.ArtifactVersionsIndex, versionIDs []string) []string {
	ids := make([]string, 0, len(versionIDs))
	seen := map[string]struct{}{}
	for _, versionID := range versionIDs {
		version, ok := index.ArtifactVersions[versionID]
		if !ok {
			continue
		}
		if _, ok := seen[version.ArtifactID]; ok {
			continue
		}
		seen[version.ArtifactID] = struct{}{}
		ids = append(ids, version.ArtifactID)
	}
	sort.Strings(ids)
	return ids
}

func spanEvents(raw any, fallbackTime string) []SpanEvent {
	rawEvents, ok := raw.([]any)
	if !ok || len(rawEvents) == 0 {
		return nil
	}
	events := make([]SpanEvent, 0, len(rawEvents))
	for _, rawEvent := range rawEvents {
		event, ok := rawEvent.(map[string]any)
		if !ok {
			continue
		}
		name := stringField(event, "event_type")
		if name == "" {
			name = "runner.event"
		}
		eventTime := stringField(event, "time")
		if eventTime == "" {
			eventTime = fallbackTime
		}
		attrs := map[string]any{
			"fbt.runner.event.type":             stringField(event, "event_type"),
			"fbt.runner.event.level":            stringField(event, "level"),
			"fbt.runner.event.message":          stringField(event, "message"),
			"fbt.runner.event.request_id":       stringField(event, "request_id"),
			"fbt.runner.event.transform_run_id": stringField(event, "transform_run_id"),
		}
		if eventAttrs, ok := event["attributes"].(map[string]any); ok {
			for key, value := range eventAttrs {
				attrs[key] = value
			}
		}
		events = append(events, SpanEvent{
			TimeUnixNano: unixNano(eventTime),
			Name:         name,
			Attributes:   attributes(attrs),
		})
	}
	return events
}

func errorSpanEvent(raw any, fallbackTime string) (SpanEvent, bool) {
	errRecord, ok := raw.(map[string]any)
	if !ok {
		return SpanEvent{}, false
	}
	return SpanEvent{
		TimeUnixNano: unixNano(fallbackTime),
		Name:         "exception",
		Attributes: attributes(map[string]any{
			"exception.type":    stringField(errRecord, "kind"),
			"exception.message": stringField(errRecord, "message"),
		}),
	}, true
}

func transformSpanName(transformID string, transform manifest.TransformResource) string {
	if transform.Name != "" {
		return "fbt transform " + transform.Name
	}
	return "fbt transform " + transformID
}

func attributes(values map[string]any) []KeyValue {
	keys := make([]string, 0, len(values))
	for key, value := range values {
		if emptyAttributeValue(value) {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]KeyValue, 0, len(keys))
	for _, key := range keys {
		result = append(result, KeyValue{Key: key, Value: anyValue(values[key])})
	}
	return result
}

func anyValue(value any) map[string]any {
	switch typed := value.(type) {
	case bool:
		return map[string]any{"boolValue": typed}
	case int:
		return map[string]any{"intValue": strconv.FormatInt(int64(typed), 10)}
	case int64:
		return map[string]any{"intValue": strconv.FormatInt(typed, 10)}
	case float64:
		if math.Trunc(typed) == typed {
			return map[string]any{"intValue": strconv.FormatInt(int64(typed), 10)}
		}
		return map[string]any{"doubleValue": typed}
	case float32:
		return map[string]any{"doubleValue": float64(typed)}
	case []string:
		values := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			values = append(values, anyValue(item))
		}
		return map[string]any{"arrayValue": map[string]any{"values": values}}
	case []any:
		values := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			values = append(values, anyValue(item))
		}
		return map[string]any{"arrayValue": map[string]any{"values": values}}
	default:
		return map[string]any{"stringValue": stringAttribute(value)}
	}
}

func stringAttribute(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprint(typed)
		}
		return string(data)
	}
}

func emptyAttributeValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return typed == ""
	case []string:
		return len(typed) == 0
	case []any:
		return len(typed) == 0
	}
	return false
}

func status(value string) Status {
	switch value {
	case "", "success", "ok", "pass", "blocked":
		return Status{Code: 1}
	default:
		return Status{Code: 2, Message: value}
	}
}

func unixNano(value string) string {
	if value == "" {
		return "0"
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return strconv.FormatInt(parsed.UnixNano(), 10)
		}
	}
	return "0"
}

func traceID(invocationID string) string {
	sum := sha256.Sum256([]byte("trace:" + invocationID))
	return hex.EncodeToString(sum[:16])
}

func spanID(value string) string {
	sum := sha256.Sum256([]byte("span:" + value))
	return hex.EncodeToString(sum[:8])
}

func stringField(record map[string]any, key string) string {
	if record == nil {
		return ""
	}
	value, ok := record[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

func stringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			result = append(result, fmt.Sprint(item))
		}
		return result
	default:
		return nil
	}
}

func projectName(m manifest.Manifest) string {
	if m.Metadata.ProjectName != "" {
		return m.Metadata.ProjectName
	}
	return "unknown"
}
