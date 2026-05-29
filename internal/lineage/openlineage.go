package lineage

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/nyuta01/fbt/internal/manifest"
	"github.com/nyuta01/fbt/internal/state"
)

const (
	OpenLineageProducer  = "https://github.com/nyuta01/fbt"
	OpenLineageSchemaURL = "https://openlineage.io/spec/1-0-0/OpenLineage.json#/definitions/RunEvent"

	fbtRunFacetSchema         = "https://schemas.fbt.dev/openlineage/fbt-run-facet/v1.json"
	fbtJobFacetSchema         = "https://schemas.fbt.dev/openlineage/fbt-job-facet/v1.json"
	fbtArtifactFacetSchema    = "https://schemas.fbt.dev/openlineage/fbt-artifact-facet/v1.json"
	fbtResourceFacetSchema    = "https://schemas.fbt.dev/openlineage/fbt-resource-facet/v1.json"
	fbtMaterialFacetSchema    = "https://schemas.fbt.dev/openlineage/fbt-material-facet/v1.json"
	fbtEvaluationsFacetSchema = "https://schemas.fbt.dev/openlineage/fbt-evaluations-facet/v1.json"
)

type OpenLineageInput struct {
	Manifest          manifest.Manifest
	Snapshot          state.Snapshot
	ArtifactVersions  state.ArtifactVersionsIndex
	EvaluationResults state.EvaluationResultsIndex
}

type RunEvent struct {
	EventType string    `json:"eventType"`
	EventTime string    `json:"eventTime"`
	Run       Run       `json:"run"`
	Job       Job       `json:"job"`
	Inputs    []Dataset `json:"inputs"`
	Outputs   []Dataset `json:"outputs"`
	Producer  string    `json:"producer"`
	SchemaURL string    `json:"schemaURL"`
}

type Run struct {
	RunID  string         `json:"runId"`
	Facets map[string]any `json:"facets,omitempty"`
}

type Job struct {
	Namespace string         `json:"namespace"`
	Name      string         `json:"name"`
	Facets    map[string]any `json:"facets,omitempty"`
}

type Dataset struct {
	Namespace string         `json:"namespace"`
	Name      string         `json:"name"`
	Facets    map[string]any `json:"facets,omitempty"`
}

func OpenLineageEvents(input OpenLineageInput) []RunEvent {
	versions := sortedArtifactVersions(input.ArtifactVersions)
	events := make([]RunEvent, 0, len(versions))
	namespace := projectNamespace(input.Manifest, input.Snapshot)

	for _, version := range versions {
		transform, ok := producerTransform(input.Manifest, version.ArtifactID)
		if !ok {
			events = append(events, orphanedRunEvent(input, namespace, version))
			continue
		}
		runIDSource := version.GeneratedBy
		if runIDSource == "" {
			runIDSource = version.VersionID
		}
		event := RunEvent{
			EventType: "COMPLETE",
			EventTime: eventTime(input.Manifest, input.Snapshot, version),
			Run: Run{
				RunID: deterministicUUID(runIDSource),
				Facets: map[string]any{
					"fbt_run": fbtRunFacet(transform, version),
				},
			},
			Job: Job{
				Namespace: namespace,
				Name:      transform.UniqueID,
				Facets: map[string]any{
					"fbt_job": fbtJobFacet(transform),
				},
			},
			Inputs:    inputDatasets(input, namespace, transform, version),
			Outputs:   []Dataset{outputDataset(input, namespace, version)},
			Producer:  OpenLineageProducer,
			SchemaURL: OpenLineageSchemaURL,
		}
		events = append(events, event)
	}
	return events
}

func orphanedRunEvent(input OpenLineageInput, namespace string, version state.ArtifactVersion) RunEvent {
	runIDSource := version.GeneratedBy
	if runIDSource == "" {
		runIDSource = version.VersionID
	}
	return RunEvent{
		EventType: "COMPLETE",
		EventTime: eventTime(input.Manifest, input.Snapshot, version),
		Run: Run{
			RunID: deterministicUUID(runIDSource),
			Facets: map[string]any{
				"fbt_run": orphanedRunFacet(version),
			},
		},
		Job: Job{
			Namespace: namespace,
			Name:      version.ArtifactID,
			Facets: map[string]any{
				"fbt_job": orphanedJobFacet(version),
			},
		},
		Inputs:    materialDatasets(namespace, version.Materials),
		Outputs:   []Dataset{outputDataset(input, namespace, version)},
		Producer:  OpenLineageProducer,
		SchemaURL: OpenLineageSchemaURL,
	}
}

func WriteOpenLineageNDJSON(w io.Writer, events []RunEvent) error {
	encoder := json.NewEncoder(w)
	for _, event := range events {
		if err := encoder.Encode(event); err != nil {
			return err
		}
	}
	return nil
}

func sortedArtifactVersions(index state.ArtifactVersionsIndex) []state.ArtifactVersion {
	versions := make([]state.ArtifactVersion, 0, len(index.ArtifactVersions))
	for _, version := range index.ArtifactVersions {
		versions = append(versions, version)
	}
	sort.Slice(versions, func(i, j int) bool {
		left := versions[i].CommittedAt
		if left == "" {
			left = versions[i].CreatedAt
		}
		right := versions[j].CommittedAt
		if right == "" {
			right = versions[j].CreatedAt
		}
		if left != right {
			return left < right
		}
		return versions[i].VersionID < versions[j].VersionID
	})
	return versions
}

func projectNamespace(m manifest.Manifest, snapshot state.Snapshot) string {
	projectName := m.Metadata.ProjectName
	if projectName == "" {
		projectName = snapshot.Metadata.ProjectName
	}
	if projectName == "" {
		projectName = "unknown"
	}
	return "fbt:" + projectName
}

func producerTransform(m manifest.Manifest, artifactID string) (manifest.TransformResource, bool) {
	for _, transform := range m.Transforms {
		for _, output := range transform.Outputs {
			if output.UniqueID == artifactID {
				return transform, true
			}
		}
	}
	return manifest.TransformResource{}, false
}

func eventTime(m manifest.Manifest, snapshot state.Snapshot, version state.ArtifactVersion) string {
	for _, candidate := range []string{
		version.CommittedAt,
		version.CreatedAt,
		m.Metadata.GeneratedAt,
		snapshot.Metadata.UpdatedAt,
	} {
		if candidate != "" {
			return candidate
		}
	}
	return time.Unix(0, 0).UTC().Format(time.RFC3339)
}

func deterministicUUID(value string) string {
	sum := sha1.Sum([]byte(value))
	b := sum[:16]
	b[6] = (b[6] & 0x0f) | 0x50
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uint32(b[0])<<24|uint32(b[1])<<16|uint32(b[2])<<8|uint32(b[3]),
		uint16(b[4])<<8|uint16(b[5]),
		uint16(b[6])<<8|uint16(b[7]),
		uint16(b[8])<<8|uint16(b[9]),
		uint64(b[10])<<40|uint64(b[11])<<32|uint64(b[12])<<24|uint64(b[13])<<16|uint64(b[14])<<8|uint64(b[15]),
	)
}

func fbtRunFacet(transform manifest.TransformResource, version state.ArtifactVersion) map[string]any {
	return compactFacet(fbtRunFacetSchema, map[string]any{
		"transform_id":        transform.UniqueID,
		"transform_name":      transform.Name,
		"transform_run_id":    version.GeneratedBy,
		"artifact_id":         version.ArtifactID,
		"artifact_version_id": version.VersionID,
	})
}

func fbtJobFacet(transform manifest.TransformResource) map[string]any {
	return compactFacet(fbtJobFacetSchema, map[string]any{
		"transform_id":   transform.UniqueID,
		"transform_name": transform.Name,
		"transform_type": transform.TransformType,
		"runner":         transform.Runner,
		"model":          transform.Model,
		"policy":         transform.Policy,
		"evals":          transform.Evals,
		"determinism":    transform.Determinism,
		"tools":          transform.Tools,
	})
}

func orphanedRunFacet(version state.ArtifactVersion) map[string]any {
	return compactFacet(fbtRunFacetSchema, map[string]any{
		"transform_run_id":    version.GeneratedBy,
		"artifact_id":         version.ArtifactID,
		"artifact_version_id": version.VersionID,
		"orphaned":            true,
	})
}

func orphanedJobFacet(version state.ArtifactVersion) map[string]any {
	return compactFacet(fbtJobFacetSchema, map[string]any{
		"artifact_id": version.ArtifactID,
		"orphaned":    true,
	})
}

func inputDatasets(input OpenLineageInput, namespace string, transform manifest.TransformResource, version state.ArtifactVersion) []Dataset {
	if len(version.Materials) > 0 {
		return materialDatasets(namespace, version.Materials)
	}

	datasets := make([]Dataset, 0, len(transform.Inputs))
	for _, transformInput := range transform.Inputs {
		facets := map[string]any{
			"fbt_resource": resourceFacet(input, transformInput),
		}
		if transformInput.Kind == "artifact" {
			if version, ok := currentArtifactVersion(input.Snapshot, input.ArtifactVersions, transformInput.UniqueID); ok {
				facets["fbt_artifact"] = artifactFacet(version)
			}
		}
		datasets = append(datasets, Dataset{
			Namespace: namespace,
			Name:      transformInput.UniqueID,
			Facets:    facets,
		})
	}
	return datasets
}

func materialDatasets(namespace string, inputMaterials []state.Material) []Dataset {
	if len(inputMaterials) == 0 {
		return nil
	}
	materials := append([]state.Material(nil), inputMaterials...)
	sort.Slice(materials, func(i, j int) bool {
		left := materials[i].ResourceID + "\x00" + materials[i].ArtifactVersion + "\x00" + materials[i].Digest
		right := materials[j].ResourceID + "\x00" + materials[j].ArtifactVersion + "\x00" + materials[j].Digest
		return left < right
	})
	datasets := make([]Dataset, 0, len(materials))
	for _, material := range materials {
		datasets = append(datasets, Dataset{
			Namespace: namespace,
			Name:      material.ResourceID,
			Facets: map[string]any{
				"fbt_material": compactFacet(fbtMaterialFacetSchema, map[string]any{
					"resource_id":         material.ResourceID,
					"artifact_version_id": material.ArtifactVersion,
					"digest":              material.Digest,
				}),
			},
		})
	}
	return datasets
}

func resourceFacet(input OpenLineageInput, transformInput manifest.TransformInput) map[string]any {
	fields := map[string]any{
		"kind":        transformInput.Kind,
		"resource_id": transformInput.UniqueID,
		"name":        transformInput.Name,
		"require":     transformInput.Require,
	}
	switch transformInput.Kind {
	case "source":
		if source, ok := input.Manifest.Sources[transformInput.UniqueID]; ok {
			fields["artifact_type"] = source.ArtifactType
			fields["path"] = source.Path
			fields["resolved_paths"] = source.ResolvedPaths
			fields["fingerprint"] = source.Fingerprint
		}
	case "artifact":
		if artifact, ok := input.Manifest.Artifacts[transformInput.UniqueID]; ok {
			fields["artifact_type"] = artifact.ArtifactType
			fields["logical_path"] = artifact.LogicalPath
		}
	}
	return compactFacet(fbtResourceFacetSchema, fields)
}

func currentArtifactVersion(snapshot state.Snapshot, index state.ArtifactVersionsIndex, artifactID string) (state.ArtifactVersion, bool) {
	pointer, ok := snapshot.CurrentArtifacts[artifactID]
	if !ok {
		return state.ArtifactVersion{}, false
	}
	version, ok := index.ArtifactVersions[pointer.CurrentVersionID]
	return version, ok
}

func outputDataset(input OpenLineageInput, namespace string, version state.ArtifactVersion) Dataset {
	facets := map[string]any{
		"fbt_artifact": artifactFacet(version),
	}
	evaluations := evaluationResults(input.EvaluationResults, version)
	if len(evaluations) > 0 {
		facets["fbt_evaluations"] = compactFacet(fbtEvaluationsFacetSchema, map[string]any{
			"results": evaluations,
		})
	}
	return Dataset{
		Namespace: namespace,
		Name:      version.ArtifactID,
		Facets:    facets,
	}
}

func artifactFacet(version state.ArtifactVersion) map[string]any {
	return compactFacet(fbtArtifactFacetSchema, map[string]any{
		"artifact_id":         version.ArtifactID,
		"artifact_version_id": version.VersionID,
		"logical_path":        version.LogicalPath,
		"storage_path":        version.StoragePath,
		"descriptor":          version.Descriptor,
		"semantic_descriptor": version.SemanticDescriptor,
		"generated_by":        version.GeneratedBy,
		"confidence":          version.Confidence,
		"created_at":          version.CreatedAt,
		"committed_at":        version.CommittedAt,
		"materials":           version.Materials,
	})
}

func evaluationResults(index state.EvaluationResultsIndex, version state.ArtifactVersion) []state.EvaluationResult {
	results := make([]state.EvaluationResult, 0)
	for _, result := range index.EvaluationResults {
		if result.ArtifactVersionID == version.VersionID || (version.GeneratedBy != "" && result.TransformRunID == version.GeneratedBy) {
			results = append(results, result)
		}
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].ResultID < results[j].ResultID
	})
	return results
}

func compactFacet(schemaURL string, fields map[string]any) map[string]any {
	facet := map[string]any{"_schemaURL": schemaURL}
	for key, value := range fields {
		if emptyFacetValue(value) {
			continue
		}
		facet[key] = value
	}
	return facet
}

func emptyFacetValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return typed == ""
	case []string:
		return len(typed) == 0
	case []state.Material:
		return len(typed) == 0
	case []state.EvaluationResult:
		return len(typed) == 0
	case map[string]any:
		return len(typed) == 0
	}
	return false
}
