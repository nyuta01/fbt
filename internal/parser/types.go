package parser

import "github.com/nyuta01/fbt/internal/config"

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

type Diagnostic struct {
	Severity Severity `json:"severity"`
	Code     string   `json:"code"`
	Message  string   `json:"message"`
	File     string   `json:"file,omitempty"`
	Line     int      `json:"line,omitempty"`
	Resource string   `json:"resource,omitempty"`
	Hint     string   `json:"hint,omitempty"`
}

type DiagnosticsError struct {
	Diagnostics []Diagnostic
}

func (e DiagnosticsError) Error() string {
	count := 0
	for _, diagnostic := range e.Diagnostics {
		if diagnostic.Severity == SeverityError {
			count++
		}
	}
	if count == 1 {
		return "project parse failed with 1 error"
	}
	return "project parse failed with multiple errors"
}

type Options struct {
	ProjectDir string
}

type Result struct {
	ProjectDir string
	ConfigPath string
	Config     config.ProjectConfig

	Sources    []Source
	Artifacts  []Artifact
	Assets     []Asset
	Transforms []Transform
	Policies   []Policy
	Evals      []Eval
	Runners    []config.RunnerConfig

	Diagnostics []Diagnostic
	lineIndex   map[string]int
}

type Source struct {
	Name        string           `yaml:"name"`
	Description string           `yaml:"description"`
	Artifacts   []SourceArtifact `yaml:"artifacts"`
	File        string           `yaml:"-"`
}

type SourceArtifact struct {
	Name        string         `yaml:"name"`
	Type        string         `yaml:"type"`
	Path        string         `yaml:"path"`
	Description string         `yaml:"description"`
	Tags        []string       `yaml:"tags"`
	Tests       []any          `yaml:"tests"`
	Meta        map[string]any `yaml:"meta"`
}

type Artifact struct {
	Name     string         `yaml:"name"`
	Type     string         `yaml:"type"`
	Path     string         `yaml:"path"`
	Contract map[string]any `yaml:"contract"`
	Owner    string         `yaml:"owner"`
	Tags     []string       `yaml:"tags"`
	Meta     map[string]any `yaml:"meta"`
	File     string         `yaml:"-"`
}

type Asset struct {
	Name      string         `yaml:"name"`
	Type      string         `yaml:"type"`
	Path      string         `yaml:"path"`
	Variables []string       `yaml:"variables"`
	Meta      map[string]any `yaml:"meta"`
	File      string         `yaml:"-"`
}

type Transform struct {
	Name     string         `yaml:"name"`
	Type     string         `yaml:"type"`
	Runner   string         `yaml:"runner"`
	Command  []string       `yaml:"command"`
	Model    map[string]any `yaml:"model"`
	Agent    string         `yaml:"agent"`
	Inputs   []Input        `yaml:"inputs"`
	Outputs  []Output       `yaml:"outputs"`
	Assets   []AssetRef     `yaml:"assets"`
	Tools    []string       `yaml:"tools"`
	Policy   string         `yaml:"policy"`
	Evals    []string       `yaml:"evals"`
	Contract map[string]any `yaml:"contract"`
	Tags     []string       `yaml:"tags"`
	Meta     map[string]any `yaml:"meta"`
	File     string         `yaml:"-"`
}

type Input struct {
	Source  string         `yaml:"source"`
	Ref     string         `yaml:"ref"`
	Require map[string]any `yaml:"require"`
}

type Output struct {
	Name     string         `yaml:"name"`
	Type     string         `yaml:"type"`
	Path     string         `yaml:"path"`
	Contract map[string]any `yaml:"contract"`
}

type AssetRef struct {
	Ref  string `yaml:"ref"`
	Type string `yaml:"type"`
	Path string `yaml:"path"`
}

type Policy struct {
	Name    string         `yaml:"name"`
	Read    []string       `yaml:"read"`
	Write   []string       `yaml:"write"`
	Network *bool          `yaml:"network"`
	Tools   map[string]any `yaml:"tools"`
	Limits  map[string]any `yaml:"limits"`
	File    string         `yaml:"-"`
}

type Eval struct {
	Name             string         `yaml:"name"`
	Type             string         `yaml:"type"`
	Runner           string         `yaml:"runner"`
	Config           map[string]any `yaml:"config"`
	GrantsConfidence string         `yaml:"grants_confidence"`
	File             string         `yaml:"-"`
}
