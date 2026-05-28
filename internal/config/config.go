package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const CurrentConfigVersion = 1

type ProjectConfig struct {
	Name           string          `yaml:"name"`
	ConfigVersion  int             `yaml:"config_version"`
	Version        string          `yaml:"version"`
	SourcePaths    []string        `yaml:"source_paths"`
	TransformPaths []string        `yaml:"transform_paths"`
	AssetPaths     []string        `yaml:"asset_paths"`
	PolicyPaths    []string        `yaml:"policy_paths"`
	EvalPaths      []string        `yaml:"eval_paths"`
	TargetPath     string          `yaml:"target_path"`
	ArtifactPath   string          `yaml:"artifact_path"`
	State          StateConfig     `yaml:"state"`
	Execution      ExecutionConfig `yaml:"execution"`
	Defaults       Defaults        `yaml:"defaults"`
	Runners        []RunnerConfig  `yaml:"runners"`
	Selectors      []Selector      `yaml:"selectors"`
	Vars           map[string]any  `yaml:"vars"`
}

type StateConfig struct {
	Backend string `yaml:"backend"`
	Path    string `yaml:"path"`
}

type ExecutionConfig struct {
	Mode       string `yaml:"mode"`
	MaxWorkers int    `yaml:"max_workers"`
	FailFast   bool   `yaml:"fail_fast"`
}

type Defaults struct {
	Review     ReviewDefault     `yaml:"review"`
	Cache      CacheDefault      `yaml:"cache"`
	Confidence ConfidenceDefault `yaml:"confidence"`
}

type ReviewDefault struct {
	Required bool   `yaml:"required"`
	Group    string `yaml:"group"`
}

type CacheDefault struct {
	Mode string `yaml:"mode"`
}

type ConfidenceDefault struct {
	Minimum string `yaml:"minimum"`
}

type RunnerConfig struct {
	Name         string         `yaml:"name"`
	Type         string         `yaml:"type"`
	Protocol     string         `yaml:"protocol"`
	Command      string         `yaml:"command"`
	Args         []string       `yaml:"args"`
	CWD          string         `yaml:"cwd"`
	Env          []string       `yaml:"env"`
	Config       map[string]any `yaml:"config"`
	Capabilities map[string]any `yaml:"capabilities"`
}

type Selector struct {
	Name       string             `yaml:"name"`
	Definition SelectorDefinition `yaml:"definition"`
}

type SelectorDefinition struct {
	Method string               `yaml:"method"`
	Value  string               `yaml:"value"`
	Union  []SelectorDefinition `yaml:"union"`
}

type VersionStatus int

const (
	VersionSupported VersionStatus = iota
	VersionMissing
	VersionUnsupported
)

type ProjectFile struct {
	Config        ProjectConfig
	VersionStatus VersionStatus
}

func LoadProjectFile(path string) (ProjectFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ProjectFile{}, err
	}
	return DecodeProjectFile(data)
}

func DecodeProjectFile(data []byte) (ProjectFile, error) {
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return ProjectFile{}, err
	}

	var cfg ProjectConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return ProjectFile{}, err
	}
	applyDraftAliases(&cfg, raw)
	cfg.ApplyDefaults()

	status := VersionSupported
	rawVersion, ok := raw["config_version"]
	if !ok {
		rawVersion, ok = raw["config-version"]
	}
	if !ok {
		status = VersionMissing
	} else if version, ok := asInt(rawVersion); !ok || version != CurrentConfigVersion {
		status = VersionUnsupported
	}

	return ProjectFile{Config: cfg, VersionStatus: status}, nil
}

func (c *ProjectConfig) ApplyDefaults() {
	if len(c.SourcePaths) == 0 {
		c.SourcePaths = []string{"sources"}
	}
	if len(c.TransformPaths) == 0 {
		c.TransformPaths = []string{"transforms"}
	}
	if len(c.AssetPaths) == 0 {
		c.AssetPaths = []string{"prompts", "assets"}
	}
	if len(c.PolicyPaths) == 0 {
		c.PolicyPaths = []string{"policies"}
	}
	if len(c.EvalPaths) == 0 {
		c.EvalPaths = []string{"evals"}
	}
	if c.TargetPath == "" {
		c.TargetPath = "target"
	}
	if c.ArtifactPath == "" {
		c.ArtifactPath = "target/artifacts"
	}
	if c.State.Backend == "" {
		c.State.Backend = "local"
	}
	if c.State.Path == "" {
		c.State.Path = ".fbt/state"
	}
	if c.Execution.Mode == "" {
		c.Execution.Mode = "local"
	}
	if c.Execution.MaxWorkers == 0 {
		c.Execution.MaxWorkers = 1
	}
	if c.Defaults.Cache.Mode == "" {
		c.Defaults.Cache.Mode = "reuse_if_same_inputs"
	}
}

func applyDraftAliases(c *ProjectConfig, raw map[string]any) {
	if c.ConfigVersion == 0 {
		if v, ok := raw["config-version"]; ok {
			if n, ok := asInt(v); ok {
				c.ConfigVersion = n
			}
		}
	}
	if len(c.SourcePaths) == 0 {
		c.SourcePaths = asStringSliceAlias(raw["source-paths"])
	}
	if len(c.TransformPaths) == 0 {
		c.TransformPaths = asStringSliceAlias(raw["transform-paths"])
	}
	if len(c.AssetPaths) == 0 {
		c.AssetPaths = asStringSliceAlias(raw["asset-paths"])
	}
	if len(c.PolicyPaths) == 0 {
		c.PolicyPaths = asStringSliceAlias(raw["policy-paths"])
	}
	if len(c.EvalPaths) == 0 {
		c.EvalPaths = asStringSliceAlias(raw["eval-paths"])
	}
	if c.TargetPath == "" {
		c.TargetPath, _ = raw["target-path"].(string)
	}
	if c.ArtifactPath == "" {
		c.ArtifactPath, _ = raw["artifact-path"].(string)
	}
}

func asStringSliceAlias(value any) []string {
	switch typed := value.(type) {
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			s, ok := item.(string)
			if !ok {
				return nil
			}
			out = append(out, s)
		}
		return out
	case []string:
		return typed
	default:
		return nil
	}
}

func asInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		if typed == float64(int(typed)) {
			return int(typed), true
		}
	}
	return 0, false
}

type ArtifactPathKind string

const (
	PathKindFile      ArtifactPathKind = "file"
	PathKindDirectory ArtifactPathKind = "directory"
)

type ArtifactType struct {
	Alias      string
	Descriptor string
	PathKind   ArtifactPathKind
}

var artifactTypes = map[string]ArtifactType{
	"text":               {Alias: "text", Descriptor: "fbt.artifact.text_file.v1", PathKind: PathKindFile},
	"markdown":           {Alias: "markdown", Descriptor: "fbt.artifact.markdown_document.v1", PathKind: PathKindFile},
	"markdown_directory": {Alias: "markdown_directory", Descriptor: "fbt.artifact.markdown_directory.v1", PathKind: PathKindDirectory},
	"docx":               {Alias: "docx", Descriptor: "fbt.artifact.docx_document.v1", PathKind: PathKindFile},
	"docx_directory":     {Alias: "docx_directory", Descriptor: "fbt.artifact.docx_directory.v1", PathKind: PathKindDirectory},
	"xlsx":               {Alias: "xlsx", Descriptor: "fbt.artifact.xlsx_workbook.v1", PathKind: PathKindFile},
	"xlsx_directory":     {Alias: "xlsx_directory", Descriptor: "fbt.artifact.xlsx_directory.v1", PathKind: PathKindDirectory},
	"pdf":                {Alias: "pdf", Descriptor: "fbt.artifact.pdf_document.v1", PathKind: PathKindFile},
	"html":               {Alias: "html", Descriptor: "fbt.artifact.html_document.v1", PathKind: PathKindFile},
	"json":               {Alias: "json", Descriptor: "fbt.artifact.json_document.v1", PathKind: PathKindFile},
	"jsonl_directory":    {Alias: "jsonl_directory", Descriptor: "fbt.artifact.jsonl_directory.v1", PathKind: PathKindDirectory},
	"directory":          {Alias: "directory", Descriptor: "fbt.artifact.directory.v1", PathKind: PathKindDirectory},
	"binary":             {Alias: "binary", Descriptor: "fbt.artifact.binary_file.v1", PathKind: PathKindFile},
}

func LookupArtifactType(alias string) (ArtifactType, bool) {
	artifactType, ok := artifactTypes[alias]
	return artifactType, ok
}

func SupportedArtifactAliases() []string {
	aliases := make([]string, 0, len(artifactTypes))
	for alias := range artifactTypes {
		aliases = append(aliases, alias)
	}
	return aliases
}

func IsCustomArtifactType(value string) bool {
	return len(value) > 2 && value[:2] == "x."
}

func ValidateArtifactType(alias string) error {
	if _, ok := LookupArtifactType(alias); ok {
		return nil
	}
	if IsCustomArtifactType(alias) {
		return nil
	}
	return fmt.Errorf("unsupported artifact type %q", alias)
}
