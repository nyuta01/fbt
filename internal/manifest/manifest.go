package manifest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nyuta01/fbt/internal/config"
	"github.com/nyuta01/fbt/internal/parser"
	versioninfo "github.com/nyuta01/fbt/internal/version"
)

const SchemaVersion = "https://schemas.fbt.dev/fbt/manifest/v1.json"

type BuildOptions struct {
	FBTVersion   string
	GeneratedAt  time.Time
	InvocationID string
	TargetName   string
}

type Manifest struct {
	Metadata         Metadata                             `json:"metadata"`
	Sources          map[string]SourceResource            `json:"sources"`
	Artifacts        map[string]ArtifactResource          `json:"artifacts"`
	ArtifactVersions map[string]any                       `json:"artifact_versions"`
	Transforms       map[string]TransformResource         `json:"transforms"`
	TransformAssets  map[string]TransformAssetResource    `json:"transform_assets"`
	Policies         map[string]PolicyResource            `json:"policies"`
	Evals            map[string]EvalResource              `json:"evals"`
	Runners          map[string]RunnerResource            `json:"runners"`
	ParentMap        map[string][]string                  `json:"parent_map"`
	ChildMap         map[string][]string                  `json:"child_map"`
	Selectors        map[string]config.SelectorDefinition `json:"selectors"`
	Disabled         map[string]any                       `json:"disabled"`
	StateSnapshot    map[string]any                       `json:"state_snapshot"`
	Files            map[string]FileResource              `json:"files"`
}

type Metadata struct {
	FBTSchemaVersion string `json:"fbt_schema_version"`
	FBTVersion       string `json:"fbt_version"`
	ProjectName      string `json:"project_name"`
	ProjectID        string `json:"project_id"`
	GeneratedAt      string `json:"generated_at"`
	InvocationID     string `json:"invocation_id,omitempty"`
	TargetName       string `json:"target_name"`
}

type Fingerprint struct {
	Method string `json:"method,omitempty"`
	Value  string `json:"value,omitempty"`
}

type SourceResource struct {
	UniqueID      string         `json:"unique_id"`
	ResourceType  string         `json:"resource_type"`
	Name          string         `json:"name"`
	SourceName    string         `json:"source_name"`
	ArtifactType  string         `json:"artifact_type"`
	Path          string         `json:"path"`
	ResolvedPaths []string       `json:"resolved_paths,omitempty"`
	Fingerprint   Fingerprint    `json:"fingerprint"`
	Tags          []string       `json:"tags,omitempty"`
	Meta          map[string]any `json:"meta,omitempty"`
}

type ArtifactResource struct {
	UniqueID     string         `json:"unique_id"`
	ResourceType string         `json:"resource_type"`
	Name         string         `json:"name"`
	ArtifactType string         `json:"artifact_type"`
	LogicalPath  string         `json:"logical_path"`
	Current      map[string]any `json:"current,omitempty"`
	Contract     map[string]any `json:"contract,omitempty"`
	Tags         []string       `json:"tags,omitempty"`
	Meta         map[string]any `json:"meta,omitempty"`
}

type TransformResource struct {
	UniqueID      string            `json:"unique_id"`
	ResourceType  string            `json:"resource_type"`
	Name          string            `json:"name"`
	TransformType string            `json:"transform_type"`
	Runner        string            `json:"runner"`
	Inputs        []TransformInput  `json:"inputs"`
	Outputs       []TransformOutput `json:"outputs"`
	Assets        []string          `json:"assets,omitempty"`
	Policy        string            `json:"policy,omitempty"`
	Evals         []string          `json:"evals,omitempty"`
	Review        map[string]any    `json:"review,omitempty"`
	Model         map[string]any    `json:"model,omitempty"`
	Tools         []string          `json:"tools,omitempty"`
	Determinism   string            `json:"determinism"`
	Cache         map[string]any    `json:"cache,omitempty"`
	Tags          []string          `json:"tags,omitempty"`
	File          string            `json:"file,omitempty"`
	Fingerprint   map[string]string `json:"fingerprint"`
}

type TransformInput struct {
	Kind     string         `json:"kind"`
	UniqueID string         `json:"unique_id"`
	Name     string         `json:"name"`
	Require  map[string]any `json:"require,omitempty"`
}

type TransformOutput struct {
	UniqueID     string `json:"unique_id"`
	Name         string `json:"name"`
	ArtifactType string `json:"artifact_type"`
	DeclaredPath string `json:"declared_path"`
}

type TransformAssetResource struct {
	UniqueID     string         `json:"unique_id"`
	ResourceType string         `json:"resource_type"`
	Name         string         `json:"name"`
	AssetType    string         `json:"asset_type"`
	Path         string         `json:"path"`
	Fingerprint  Fingerprint    `json:"fingerprint"`
	Variables    []string       `json:"variables,omitempty"`
	Tags         []string       `json:"tags,omitempty"`
	Meta         map[string]any `json:"meta,omitempty"`
}

type PolicyResource struct {
	UniqueID     string         `json:"unique_id"`
	ResourceType string         `json:"resource_type"`
	Name         string         `json:"name"`
	Fingerprint  Fingerprint    `json:"fingerprint"`
	ReadScope    []string       `json:"read_scope,omitempty"`
	WriteScope   []string       `json:"write_scope,omitempty"`
	Network      *bool          `json:"network,omitempty"`
	Tools        map[string]any `json:"tools,omitempty"`
	Limits       map[string]any `json:"limits,omitempty"`
	Review       map[string]any `json:"review,omitempty"`
}

type EvalResource struct {
	UniqueID         string         `json:"unique_id"`
	ResourceType     string         `json:"resource_type"`
	Name             string         `json:"name"`
	EvalType         string         `json:"eval_type"`
	Runner           string         `json:"runner,omitempty"`
	Fingerprint      Fingerprint    `json:"fingerprint"`
	Config           map[string]any `json:"config,omitempty"`
	GrantsConfidence string         `json:"grants_confidence,omitempty"`
}

type RunnerResource struct {
	UniqueID     string         `json:"unique_id"`
	ResourceType string         `json:"resource_type"`
	Name         string         `json:"name"`
	RunnerType   string         `json:"runner_type"`
	Protocol     string         `json:"protocol"`
	Command      string         `json:"command,omitempty"`
	Env          []string       `json:"env,omitempty"`
	Config       map[string]any `json:"config,omitempty"`
	Capabilities map[string]any `json:"capabilities,omitempty"`
	Fingerprint  Fingerprint    `json:"fingerprint"`
}

type FileResource struct {
	Path        string   `json:"path"`
	Checksum    string   `json:"checksum"`
	ResourceIDs []string `json:"resource_ids"`
}

type ResourceSummary struct {
	UniqueID     string
	ResourceType string
	Name         string
	Path         string
	Tags         []string
}

func Build(parseResult parser.Result, options BuildOptions) (Manifest, error) {
	if options.FBTVersion == "" {
		options.FBTVersion = versioninfo.Version
	}
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	if options.TargetName == "" {
		options.TargetName = "local"
	}

	builder := manifestBuilder{
		project: parseResult.Config.Name,
		root:    parseResult.ProjectDir,
		manifest: Manifest{
			Metadata: Metadata{
				FBTSchemaVersion: SchemaVersion,
				FBTVersion:       options.FBTVersion,
				ProjectName:      parseResult.Config.Name,
				ProjectID:        hashString("project:" + parseResult.Config.Name),
				GeneratedAt:      options.GeneratedAt.UTC().Format(time.RFC3339),
				InvocationID:     options.InvocationID,
				TargetName:       options.TargetName,
			},
			Sources:          map[string]SourceResource{},
			Artifacts:        map[string]ArtifactResource{},
			ArtifactVersions: map[string]any{},
			Transforms:       map[string]TransformResource{},
			TransformAssets:  map[string]TransformAssetResource{},
			Policies:         map[string]PolicyResource{},
			Evals:            map[string]EvalResource{},
			Runners:          map[string]RunnerResource{},
			ParentMap:        map[string][]string{},
			ChildMap:         map[string][]string{},
			Selectors:        map[string]config.SelectorDefinition{},
			Disabled:         map[string]any{},
			StateSnapshot:    map[string]any{},
			Files:            map[string]FileResource{},
		},
	}

	for _, selector := range parseResult.Config.Selectors {
		builder.manifest.Selectors[selector.Name] = selector.Definition
	}
	for _, source := range parseResult.Sources {
		builder.addSource(source)
	}
	for _, artifact := range parseResult.Artifacts {
		builder.addArtifact(artifact.Name, artifact.Type, artifact.Path, artifact.Contract, artifact.Tags, artifact.Meta)
	}
	for _, transform := range parseResult.Transforms {
		for _, output := range transform.Outputs {
			builder.addArtifact(output.Name, output.Type, output.Path, output.Contract, transform.Tags, nil)
		}
	}
	for _, asset := range parseResult.Assets {
		builder.addAsset(asset.Name, asset.Type, asset.Path, asset.Variables, asset.Meta)
	}
	for _, policy := range parseResult.Policies {
		builder.addPolicy(policy)
	}
	for _, eval := range parseResult.Evals {
		builder.addEval(eval)
	}
	for _, runner := range parseResult.Runners {
		builder.addRunner(runner)
	}
	for _, transform := range parseResult.Transforms {
		builder.addTransform(transform)
	}

	builder.sortGraph()
	return builder.manifest, nil
}

type manifestBuilder struct {
	project  string
	root     string
	manifest Manifest
}

func (b *manifestBuilder) addSource(source parser.Source) {
	for _, artifact := range source.Artifacts {
		id := SourceID(b.project, source.Name, artifact.Name)
		resolved := b.resolvePaths(artifact.Path)
		b.manifest.Sources[id] = SourceResource{
			UniqueID:      id,
			ResourceType:  "source",
			Name:          artifact.Name,
			SourceName:    source.Name,
			ArtifactType:  artifact.Type,
			Path:          slashPath(artifact.Path),
			ResolvedPaths: resolved,
			Fingerprint:   Fingerprint{Method: "definition", Value: hashJSON(artifact)},
			Tags:          sortedCopy(artifact.Tags),
			Meta:          artifact.Meta,
		}
	}
}

func (b *manifestBuilder) addArtifact(name, artifactType, path string, contract map[string]any, tags []string, meta map[string]any) {
	id := ArtifactID(b.project, name)
	if existing, ok := b.manifest.Artifacts[id]; ok {
		if len(existing.Contract) == 0 && len(contract) > 0 {
			existing.Contract = contract
		}
		existing.Tags = sortedUnion(existing.Tags, tags)
		b.manifest.Artifacts[id] = existing
		return
	}
	b.manifest.Artifacts[id] = ArtifactResource{
		UniqueID:     id,
		ResourceType: "artifact",
		Name:         name,
		ArtifactType: artifactType,
		LogicalPath:  slashPath(path),
		Contract:     contract,
		Tags:         sortedCopy(tags),
		Meta:         meta,
	}
}

func (b *manifestBuilder) addAsset(name, assetType, path string, variables []string, meta map[string]any) string {
	id := TransformAssetID(b.project, name)
	b.manifest.TransformAssets[id] = TransformAssetResource{
		UniqueID:     id,
		ResourceType: "transform_asset",
		Name:         name,
		AssetType:    assetType,
		Path:         slashPath(path),
		Fingerprint:  b.fileFingerprint(path),
		Variables:    sortedCopy(variables),
		Meta:         meta,
	}
	b.addFile(path, id)
	return id
}

func (b *manifestBuilder) addPolicy(policy parser.Policy) {
	id := PolicyID(b.project, policy.Name)
	b.manifest.Policies[id] = PolicyResource{
		UniqueID:     id,
		ResourceType: "policy",
		Name:         policy.Name,
		Fingerprint:  Fingerprint{Method: "config", Value: hashJSON(policy)},
		ReadScope:    sortedSlashCopy(policy.Read),
		WriteScope:   sortedSlashCopy(policy.Write),
		Network:      policy.Network,
		Tools:        policy.Tools,
		Limits:       policy.Limits,
		Review:       policy.Review,
	}
}

func (b *manifestBuilder) addEval(eval parser.Eval) {
	id := EvalID(b.project, eval.Name)
	runnerID := ""
	if eval.Runner != "" {
		runnerID = RunnerID(b.project, eval.Runner)
	}
	b.manifest.Evals[id] = EvalResource{
		UniqueID:         id,
		ResourceType:     "eval",
		Name:             eval.Name,
		EvalType:         eval.Type,
		Runner:           runnerID,
		Fingerprint:      Fingerprint{Method: "config", Value: hashJSON(eval)},
		Config:           eval.Config,
		GrantsConfidence: eval.GrantsConfidence,
	}
}

func (b *manifestBuilder) addRunner(runner config.RunnerConfig) {
	id := RunnerID(b.project, runner.Name)
	b.manifest.Runners[id] = RunnerResource{
		UniqueID:     id,
		ResourceType: "runner",
		Name:         runner.Name,
		RunnerType:   runner.Type,
		Protocol:     runner.Protocol,
		Command:      runner.Command,
		Env:          sortedCopy(runner.Env),
		Config:       runner.Config,
		Capabilities: runner.Capabilities,
		Fingerprint:  Fingerprint{Method: "identity", Value: hashJSON(runner)},
	}
}

func (b *manifestBuilder) addTransform(transform parser.Transform) {
	id := TransformID(b.project, transform.Name)
	parents := map[string]struct{}{}

	inputs := make([]TransformInput, 0, len(transform.Inputs))
	for _, input := range transform.Inputs {
		if input.Source != "" {
			sourceName, artifactName, _ := strings.Cut(input.Source, ".")
			sourceID := SourceID(b.project, sourceName, artifactName)
			inputs = append(inputs, TransformInput{Kind: "source", UniqueID: sourceID, Name: input.Source, Require: input.Require})
			parents[sourceID] = struct{}{}
			continue
		}
		artifactID := ArtifactID(b.project, input.Ref)
		inputs = append(inputs, TransformInput{Kind: "ref", UniqueID: artifactID, Name: input.Ref, Require: input.Require})
		parents[artifactID] = struct{}{}
	}

	outputs := make([]TransformOutput, 0, len(transform.Outputs))
	for _, output := range transform.Outputs {
		artifactID := ArtifactID(b.project, output.Name)
		outputs = append(outputs, TransformOutput{
			UniqueID:     artifactID,
			Name:         output.Name,
			ArtifactType: output.Type,
			DeclaredPath: slashPath(output.Path),
		})
		b.addEdge(id, artifactID)
	}

	assets := make([]string, 0, len(transform.Assets))
	for i, asset := range transform.Assets {
		var assetID string
		if asset.Ref != "" {
			assetID = TransformAssetID(b.project, asset.Ref)
		} else {
			name := fmt.Sprintf("%s_asset_%d", transform.Name, i+1)
			assetID = b.addAsset(name, asset.Type, asset.Path, nil, nil)
		}
		assets = append(assets, assetID)
		parents[assetID] = struct{}{}
	}
	sort.Strings(assets)

	policyID := ""
	if transform.Policy != "" {
		policyID = PolicyID(b.project, transform.Policy)
		parents[policyID] = struct{}{}
	}

	evals := make([]string, 0, len(transform.Evals))
	for _, evalName := range transform.Evals {
		evalID := EvalID(b.project, evalName)
		evals = append(evals, evalID)
		parents[evalID] = struct{}{}
	}
	sort.Strings(evals)

	runnerID := ""
	if transform.Runner != "" {
		runnerID = RunnerID(b.project, transform.Runner)
		parents[runnerID] = struct{}{}
	}

	b.manifest.Transforms[id] = TransformResource{
		UniqueID:      id,
		ResourceType:  "transform",
		Name:          transform.Name,
		TransformType: transform.Type,
		Runner:        runnerID,
		Inputs:        inputs,
		Outputs:       outputs,
		Assets:        assets,
		Policy:        policyID,
		Evals:         evals,
		Review:        transform.Review,
		Model:         transform.Model,
		Tools:         sortedCopy(transform.Tools),
		Determinism:   determinism(transform.Type),
		Cache:         transform.Cache,
		Tags:          sortedCopy(transform.Tags),
		File:          slashRel(b.root, transform.File),
		Fingerprint: map[string]string{
			"config":    hashJSON(transform),
			"effective": hashJSON(map[string]any{"transform": transform, "parents": sortedKeys(parents)}),
		},
	}
	for parent := range parents {
		b.addEdge(parent, id)
	}
}

func (b *manifestBuilder) addEdge(parent, child string) {
	b.manifest.ParentMap[child] = appendUnique(b.manifest.ParentMap[child], parent)
	b.manifest.ChildMap[parent] = appendUnique(b.manifest.ChildMap[parent], child)
}

func (b *manifestBuilder) addFile(path, resourceID string) {
	if path == "" {
		return
	}
	key := slashPath(path)
	file := b.manifest.Files[key]
	file.Path = key
	file.Checksum = b.fileFingerprint(path).Value
	file.ResourceIDs = appendUnique(file.ResourceIDs, resourceID)
	b.manifest.Files[key] = file
}

func (b *manifestBuilder) fileFingerprint(path string) Fingerprint {
	if path == "" {
		return Fingerprint{Method: "content", Value: ""}
	}
	data, err := os.ReadFile(filepath.Join(b.root, filepath.Clean(path)))
	if err != nil {
		return Fingerprint{Method: "content", Value: hashString(path)}
	}
	sum := sha256.Sum256(data)
	return Fingerprint{Method: "content", Value: "sha256:" + hex.EncodeToString(sum[:])}
}

func (b *manifestBuilder) resolvePaths(path string) []string {
	if strings.Contains(path, "://") {
		return nil
	}
	clean := filepath.Clean(path)
	var paths []string
	if strings.ContainsAny(clean, "*?[") {
		matches, _ := filepath.Glob(filepath.Join(b.root, clean))
		for _, match := range matches {
			paths = append(paths, slashRel(b.root, match))
		}
		sort.Strings(paths)
		return paths
	}
	abs := filepath.Join(b.root, clean)
	info, err := os.Stat(abs)
	if err != nil {
		return nil
	}
	if !info.IsDir() {
		return []string{slashPath(clean)}
	}
	_ = filepath.WalkDir(abs, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		paths = append(paths, slashRel(b.root, path))
		return nil
	})
	sort.Strings(paths)
	return paths
}

func (b *manifestBuilder) sortGraph() {
	for key, values := range b.manifest.ParentMap {
		sort.Strings(values)
		b.manifest.ParentMap[key] = values
	}
	for key, values := range b.manifest.ChildMap {
		sort.Strings(values)
		b.manifest.ChildMap[key] = values
	}
	for key, file := range b.manifest.Files {
		sort.Strings(file.ResourceIDs)
		b.manifest.Files[key] = file
	}
}

func (m Manifest) JSON() ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}

func (m Manifest) ResourceSummaries() map[string]ResourceSummary {
	summaries := map[string]ResourceSummary{}
	for id, resource := range m.Sources {
		summaries[id] = ResourceSummary{UniqueID: id, ResourceType: resource.ResourceType, Name: resource.SourceName + "." + resource.Name, Path: resource.Path, Tags: resource.Tags}
	}
	for id, resource := range m.Artifacts {
		summaries[id] = ResourceSummary{UniqueID: id, ResourceType: resource.ResourceType, Name: resource.Name, Path: resource.LogicalPath, Tags: resource.Tags}
	}
	for id, resource := range m.Transforms {
		summaries[id] = ResourceSummary{UniqueID: id, ResourceType: resource.ResourceType, Name: resource.Name, Path: resource.File, Tags: resource.Tags}
	}
	for id, resource := range m.TransformAssets {
		summaries[id] = ResourceSummary{UniqueID: id, ResourceType: resource.ResourceType, Name: resource.Name, Path: resource.Path, Tags: resource.Tags}
	}
	for id, resource := range m.Policies {
		summaries[id] = ResourceSummary{UniqueID: id, ResourceType: resource.ResourceType, Name: resource.Name}
	}
	for id, resource := range m.Evals {
		summaries[id] = ResourceSummary{UniqueID: id, ResourceType: resource.ResourceType, Name: resource.Name}
	}
	for id, resource := range m.Runners {
		summaries[id] = ResourceSummary{UniqueID: id, ResourceType: resource.ResourceType, Name: resource.Name}
	}
	return summaries
}

func SourceID(projectName, sourceName, artifactName string) string {
	return fmt.Sprintf("source.%s.%s.%s", projectName, sourceName, artifactName)
}

func ArtifactID(projectName, name string) string {
	return fmt.Sprintf("artifact.%s.%s", projectName, name)
}

func TransformID(projectName, name string) string {
	return fmt.Sprintf("transform.%s.%s", projectName, name)
}

func TransformAssetID(projectName, name string) string {
	return fmt.Sprintf("transform_asset.%s.%s", projectName, name)
}

func PolicyID(projectName, name string) string {
	return fmt.Sprintf("policy.%s.%s", projectName, name)
}

func EvalID(projectName, name string) string {
	return fmt.Sprintf("eval.%s.%s", projectName, name)
}

func RunnerID(projectName, name string) string {
	return fmt.Sprintf("runner.%s.%s", projectName, name)
}

func determinism(transformType string) string {
	switch transformType {
	case "llm", "agent":
		return "stochastic"
	default:
		return "deterministic"
	}
}

func hashString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func hashJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return hashString(fmt.Sprintf("%#v", value))
	}
	return hashString(string(data))
}

func slashPath(path string) string {
	return filepath.ToSlash(filepath.Clean(path))
}

func slashRel(root, path string) string {
	if path == "" {
		return ""
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return slashPath(path)
	}
	return slashPath(rel)
}

func sortedCopy(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func sortedSlashCopy(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, slashPath(value))
	}
	sort.Strings(out)
	return out
}

func sortedUnion(left, right []string) []string {
	seen := map[string]struct{}{}
	for _, value := range left {
		seen[value] = struct{}{}
	}
	for _, value := range right {
		seen[value] = struct{}{}
	}
	return sortedKeys(seen)
}

func sortedKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
