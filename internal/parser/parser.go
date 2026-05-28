package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/nyuta01/fbt/internal/config"
	"github.com/nyuta01/fbt/internal/project"
	"github.com/nyuta01/fbt/internal/security"
	"gopkg.in/yaml.v3"
)

var resourceNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

var transformTypes = map[string]struct{}{
	"command":  {},
	"extract":  {},
	"template": {},
	"llm":      {},
	"agent":    {},
	"compose":  {},
}

var evalTypes = map[string]struct{}{
	"deterministic": {},
	"semantic":      {},
	"llm_judge":     {},
}

type resourceFile struct {
	Sources    []Source              `yaml:"sources"`
	Artifacts  []Artifact            `yaml:"artifacts"`
	Assets     []Asset               `yaml:"assets"`
	Transforms []Transform           `yaml:"transforms"`
	Policies   []Policy              `yaml:"policies"`
	Evals      []Eval                `yaml:"evals"`
	Runners    []config.RunnerConfig `yaml:"runners"`
}

func ParseProject(options Options) (Result, error) {
	prj, err := project.Open(options.ProjectDir)
	if err != nil {
		result := Result{}
		result.addError("PROJECT_NOT_FOUND", err.Error(), "", "")
		return result, DiagnosticsError{Diagnostics: result.Diagnostics}
	}

	result := Result{
		ProjectDir: prj.RootDir,
		ConfigPath: prj.ConfigPath,
		lineIndex:  map[string]int{},
	}

	projectFile, err := config.LoadProjectFile(prj.ConfigPath)
	if err != nil {
		result.addError("CONFIG_READ_FAILED", err.Error(), prj.ConfigPath, "")
		return result, DiagnosticsError{Diagnostics: result.Diagnostics}
	}
	result.Config = projectFile.Config
	result.Runners = append(result.Runners, projectFile.Config.Runners...)

	switch projectFile.VersionStatus {
	case config.VersionMissing:
		result.addError("CONFIG_VERSION_MISSING", "fs_project.yml must include config_version: 1", prj.ConfigPath, "")
	case config.VersionUnsupported:
		result.addError("CONFIG_VERSION_UNSUPPORTED", fmt.Sprintf("unsupported config_version %d; expected %d", projectFile.Config.ConfigVersion, config.CurrentConfigVersion), prj.ConfigPath, "")
	}
	if result.hasErrors() {
		return result, DiagnosticsError{Diagnostics: result.Diagnostics}
	}

	if result.Config.Name == "" {
		result.addError("PROJECT_NAME_REQUIRED", "project name is required", prj.ConfigPath, "")
	} else {
		result.validateName("PROJECT_NAME_INVALID", "project", result.Config.Name, prj.ConfigPath)
	}
	result.validateProjectPath("ARTIFACT_PATH_INVALID", result.Config.ArtifactPath, prj.ConfigPath, "project")
	result.validateProjectPath("TARGET_PATH_INVALID", result.Config.TargetPath, prj.ConfigPath, "project")

	files, err := resourceFiles(prj.RootDir, result.resourceDirs())
	if err != nil {
		result.addError("RESOURCE_DISCOVERY_FAILED", err.Error(), prj.RootDir, "")
		return result, DiagnosticsError{Diagnostics: result.Diagnostics}
	}

	for _, file := range files {
		loaded, lines, unsupported, err := loadResourceFile(file)
		if err != nil {
			result.addError("RESOURCE_PARSE_FAILED", err.Error(), file, "")
			continue
		}
		result.appendResources(file, loaded, lines)
		for _, field := range unsupported {
			result.addUnsupportedField(file, field)
		}
	}

	result.validateResources()
	if result.hasErrors() {
		return result, DiagnosticsError{Diagnostics: result.Diagnostics}
	}

	return result, nil
}

func (r *Result) resourceDirs() []string {
	var dirs []string
	dirs = append(dirs, r.Config.SourcePaths...)
	dirs = append(dirs, r.Config.TransformPaths...)
	dirs = append(dirs, r.Config.AssetPaths...)
	dirs = append(dirs, r.Config.PolicyPaths...)
	dirs = append(dirs, r.Config.EvalPaths...)
	return dirs
}

func resourceFiles(root string, dirs []string) ([]string, error) {
	seenDirs := map[string]struct{}{}
	seenFiles := map[string]struct{}{}
	var files []string

	for _, dir := range dirs {
		clean, err := cleanProjectRelativePath(dir)
		if err != nil {
			return nil, fmt.Errorf("invalid resource directory %q: %w", dir, err)
		}
		abs := filepath.Join(root, clean)
		if _, ok := seenDirs[abs]; ok {
			continue
		}
		seenDirs[abs] = struct{}{}

		info, err := os.Stat(abs)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			continue
		}

		if err := filepath.WalkDir(abs, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".yml" && ext != ".yaml" {
				return nil
			}
			if _, ok := seenFiles[path]; !ok {
				seenFiles[path] = struct{}{}
				files = append(files, path)
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}

	sort.Strings(files)
	return files, nil
}

type unsupportedField struct {
	code     string
	message  string
	resource string
	line     int
}

func loadResourceFile(path string) (resourceFile, map[string]int, []unsupportedField, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return resourceFile{}, nil, nil, err
	}
	var loaded resourceFile
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		return resourceFile{}, nil, nil, err
	}
	return loaded, resourceLineIndex(data), unsupportedReviewFields(data), nil
}

func (r *Result) appendResources(file string, loaded resourceFile, lines map[string]int) {
	if r.lineIndex == nil {
		r.lineIndex = map[string]int{}
	}
	for key, line := range lines {
		r.lineIndex[diagKey(file, key)] = line
	}
	for i := range loaded.Sources {
		loaded.Sources[i].File = file
	}
	for i := range loaded.Artifacts {
		loaded.Artifacts[i].File = file
	}
	for i := range loaded.Assets {
		loaded.Assets[i].File = file
	}
	for i := range loaded.Transforms {
		loaded.Transforms[i].File = file
	}
	for i := range loaded.Policies {
		loaded.Policies[i].File = file
	}
	for i := range loaded.Evals {
		loaded.Evals[i].File = file
	}
	r.Sources = append(r.Sources, loaded.Sources...)
	r.Artifacts = append(r.Artifacts, loaded.Artifacts...)
	r.Assets = append(r.Assets, loaded.Assets...)
	r.Transforms = append(r.Transforms, loaded.Transforms...)
	r.Policies = append(r.Policies, loaded.Policies...)
	r.Evals = append(r.Evals, loaded.Evals...)
	r.Runners = append(r.Runners, loaded.Runners...)
}

func resourceLineIndex(data []byte) map[string]int {
	lines := map[string]int{}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil || len(doc.Content) == 0 {
		return lines
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return lines
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		key := root.Content[i]
		value := root.Content[i+1]
		if value.Kind != yaml.SequenceNode {
			continue
		}
		for _, item := range value.Content {
			name, nameLine := mappingScalar(item, "name")
			if name == "" {
				continue
			}
			if nameLine == 0 {
				nameLine = item.Line
			}
			lines[name] = nameLine
			if key.Value == "sources" {
				indexSourceArtifacts(lines, name, item)
			}
		}
	}
	return lines
}

func indexSourceArtifacts(lines map[string]int, sourceName string, sourceNode *yaml.Node) {
	artifacts := mappingNode(sourceNode, "artifacts")
	if artifacts == nil || artifacts.Kind != yaml.SequenceNode {
		return
	}
	for _, artifactNode := range artifacts.Content {
		name, line := mappingScalar(artifactNode, "name")
		if name == "" {
			continue
		}
		if line == 0 {
			line = artifactNode.Line
		}
		lines[sourceName+"."+name] = line
	}
}

func unsupportedReviewFields(data []byte) []unsupportedField {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil || len(doc.Content) == 0 {
		return nil
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil
	}
	var fields []unsupportedField
	for i := 0; i+1 < len(root.Content); i += 2 {
		key := root.Content[i]
		value := root.Content[i+1]
		if value.Kind != yaml.SequenceNode {
			continue
		}
		switch key.Value {
		case "transforms":
			for _, item := range value.Content {
				resource, _ := mappingScalar(item, "name")
				if review := mappingNode(item, "review"); review != nil {
					fields = append(fields, unsupportedReviewField(resource, review.Line))
				}
				inputs := mappingNode(item, "inputs")
				if inputs == nil || inputs.Kind != yaml.SequenceNode {
					continue
				}
				for _, input := range inputs.Content {
					require := mappingNode(input, "require")
					if require == nil {
						continue
					}
					if review := mappingNode(require, "review"); review != nil {
						fields = append(fields, unsupportedReviewField(resource, review.Line))
					}
				}
			}
		case "policies":
			for _, item := range value.Content {
				resource, _ := mappingScalar(item, "name")
				if review := mappingNode(item, "review"); review != nil {
					fields = append(fields, unsupportedReviewField(resource, review.Line))
				}
			}
		}
	}
	return fields
}

func unsupportedReviewField(resource string, line int) unsupportedField {
	return unsupportedField{
		code:     "REVIEW_UNSUPPORTED",
		message:  "`review` is no longer part of fbt core",
		resource: resource,
		line:     line,
	}
}

func mappingScalar(node *yaml.Node, key string) (string, int) {
	value := mappingNode(node, key)
	if value == nil {
		return "", 0
	}
	return value.Value, value.Line
}

func mappingNode(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func (r *Result) validateResources() {
	sourceIDs := map[string]SourceArtifact{}
	artifactDefs := map[string]Artifact{}
	outputOwners := map[string]string{}
	assets := map[string]Asset{}
	policies := map[string]Policy{}
	evals := map[string]Eval{}
	runners := map[string]config.RunnerConfig{}

	for _, runner := range r.Runners {
		if runner.Name == "" {
			r.addError("RUNNER_NAME_REQUIRED", "runner name is required", r.ConfigPath, "runner")
			continue
		}
		if _, exists := runners[runner.Name]; exists {
			r.addError("RUNNER_DUPLICATE", fmt.Sprintf("duplicate runner %q", runner.Name), r.ConfigPath, runner.Name)
			continue
		}
		runners[runner.Name] = runner
	}

	for _, source := range r.Sources {
		r.validateName("SOURCE_NAME_INVALID", "source", source.Name, source.File)
		if source.Name == "" {
			r.addError("SOURCE_NAME_REQUIRED", "source name is required", source.File, "source")
			continue
		}
		if len(source.Artifacts) == 0 {
			r.addError("SOURCE_ARTIFACTS_REQUIRED", fmt.Sprintf("source %q must declare artifacts", source.Name), source.File, source.Name)
		}
		for _, artifact := range source.Artifacts {
			r.validateName("SOURCE_ARTIFACT_NAME_INVALID", "source artifact", artifact.Name, source.File)
			resource := source.Name + "." + artifact.Name
			if _, exists := sourceIDs[resource]; exists {
				r.addError("SOURCE_DUPLICATE", fmt.Sprintf("duplicate source artifact %q", resource), source.File, resource)
				continue
			}
			sourceIDs[resource] = artifact
			r.validateArtifactType(artifact.Type, source.File, resource)
			r.validateSourcePath(artifact.Path, source.File, resource)
		}
	}

	for _, artifact := range r.Artifacts {
		r.validateArtifactDefinition(artifact, artifact.File, artifactDefs)
	}

	for _, transform := range r.Transforms {
		for _, output := range transform.Outputs {
			artifact := Artifact{Name: output.Name, Type: output.Type, Path: output.Path, Contract: output.Contract, File: transform.File}
			r.validateArtifactDefinition(artifact, transform.File, artifactDefs)
			if owner, exists := outputOwners[output.Name]; exists && owner != transform.Name {
				r.addError("OUTPUT_DUPLICATE", fmt.Sprintf("artifact %q is declared by multiple transforms", output.Name), transform.File, output.Name)
			}
			outputOwners[output.Name] = transform.Name
		}
	}

	for _, asset := range r.Assets {
		r.validateName("ASSET_NAME_INVALID", "asset", asset.Name, asset.File)
		if asset.Name == "" {
			r.addError("ASSET_NAME_REQUIRED", "asset name is required", asset.File, "asset")
			continue
		}
		if _, exists := assets[asset.Name]; exists {
			r.addError("ASSET_DUPLICATE", fmt.Sprintf("duplicate asset %q", asset.Name), asset.File, asset.Name)
			continue
		}
		assets[asset.Name] = asset
		if asset.Path == "" {
			r.addError("ASSET_PATH_REQUIRED", fmt.Sprintf("asset %q must declare path", asset.Name), asset.File, asset.Name)
		} else {
			r.validateExistingProjectPath("ASSET_PATH_MISSING", asset.Path, asset.File, asset.Name)
		}
	}

	for _, policy := range r.Policies {
		r.validateName("POLICY_NAME_INVALID", "policy", policy.Name, policy.File)
		if policy.Name == "" {
			r.addError("POLICY_NAME_REQUIRED", "policy name is required", policy.File, "policy")
			continue
		}
		if _, exists := policies[policy.Name]; exists {
			r.addError("POLICY_DUPLICATE", fmt.Sprintf("duplicate policy %q", policy.Name), policy.File, policy.Name)
			continue
		}
		policies[policy.Name] = policy
		for _, path := range append(policy.Read, policy.Write...) {
			r.validateProjectPath("POLICY_PATH_INVALID", path, policy.File, policy.Name)
		}
	}

	for _, eval := range r.Evals {
		r.validateName("EVAL_NAME_INVALID", "eval", eval.Name, eval.File)
		if eval.Name == "" {
			r.addError("EVAL_NAME_REQUIRED", "eval name is required", eval.File, "eval")
			continue
		}
		if _, exists := evals[eval.Name]; exists {
			r.addError("EVAL_DUPLICATE", fmt.Sprintf("duplicate eval %q", eval.Name), eval.File, eval.Name)
			continue
		}
		evals[eval.Name] = eval
		if _, ok := evalTypes[eval.Type]; !ok {
			r.addError("EVAL_TYPE_UNSUPPORTED", fmt.Sprintf("unsupported eval type %q", eval.Type), eval.File, eval.Name)
		}
	}

	for _, transform := range r.Transforms {
		r.validateTransform(transform, sourceIDs, artifactDefs, assets, policies, evals, runners)
	}
}

func (r *Result) validateArtifactDefinition(artifact Artifact, file string, artifactDefs map[string]Artifact) {
	r.validateName("ARTIFACT_NAME_INVALID", "artifact", artifact.Name, file)
	if artifact.Name == "" {
		r.addError("ARTIFACT_NAME_REQUIRED", "artifact name is required", file, "artifact")
		return
	}
	r.validateArtifactType(artifact.Type, file, artifact.Name)
	r.validateArtifactPath(artifact.Path, file, artifact.Name)
	if existing, exists := artifactDefs[artifact.Name]; exists {
		if existing.Type != artifact.Type || cleanPath(existing.Path) != cleanPath(artifact.Path) {
			r.addError("ARTIFACT_CONFLICT", fmt.Sprintf("artifact %q is declared with conflicting type or path", artifact.Name), file, artifact.Name)
		}
		return
	}
	artifactDefs[artifact.Name] = artifact
}

func (r *Result) validateTransform(transform Transform, sourceIDs map[string]SourceArtifact, artifactDefs map[string]Artifact, assets map[string]Asset, policies map[string]Policy, evals map[string]Eval, runners map[string]config.RunnerConfig) {
	r.validateName("TRANSFORM_NAME_INVALID", "transform", transform.Name, transform.File)
	if transform.Name == "" {
		r.addError("TRANSFORM_NAME_REQUIRED", "transform name is required", transform.File, "transform")
	}
	if _, ok := transformTypes[transform.Type]; !ok {
		r.addError("TRANSFORM_TYPE_UNSUPPORTED", fmt.Sprintf("unsupported transform type %q", transform.Type), transform.File, transform.Name)
	}
	if transform.Type == "command" && len(transform.Command) == 0 {
		r.addError("COMMAND_TRANSFORM_COMMAND_REQUIRED", fmt.Sprintf("command transform %q must declare command", transform.Name), transform.File, transform.Name)
	}
	if transform.Runner == "" {
		r.addError("TRANSFORM_RUNNER_REQUIRED", fmt.Sprintf("transform %q must declare runner", transform.Name), transform.File, transform.Name)
	} else if len(runners) > 0 {
		if _, ok := runners[transform.Runner]; !ok {
			r.addWarning("RUNNER_UNDECLARED", fmt.Sprintf("runner %q is not declared in project config or resource files", transform.Runner), transform.File, transform.Name)
		}
	}
	if len(transform.Inputs) == 0 {
		r.addError("TRANSFORM_INPUTS_REQUIRED", fmt.Sprintf("transform %q must declare inputs", transform.Name), transform.File, transform.Name)
	}
	for _, input := range transform.Inputs {
		r.validateInput(transform, input, sourceIDs, artifactDefs)
	}
	if len(transform.Outputs) == 0 {
		r.addError("TRANSFORM_OUTPUTS_REQUIRED", fmt.Sprintf("transform %q must declare outputs", transform.Name), transform.File, transform.Name)
	}
	for _, asset := range transform.Assets {
		if asset.Ref != "" {
			if _, ok := assets[asset.Ref]; !ok {
				r.addError("ASSET_REF_UNRESOLVED", fmt.Sprintf("transform %q references missing asset %q", transform.Name, asset.Ref), transform.File, transform.Name)
			}
			continue
		}
		if asset.Path != "" {
			r.validateExistingProjectPath("ASSET_PATH_MISSING", asset.Path, transform.File, transform.Name)
			continue
		}
		r.addError("ASSET_REFERENCE_INVALID", fmt.Sprintf("transform %q asset entry must use ref or path", transform.Name), transform.File, transform.Name)
	}
	if transform.Policy != "" {
		if _, ok := policies[transform.Policy]; !ok {
			r.addError("POLICY_REF_UNRESOLVED", fmt.Sprintf("transform %q references missing policy %q", transform.Name, transform.Policy), transform.File, transform.Name)
		}
	} else if transform.Type == "agent" {
		r.addWarning("AGENT_POLICY_MISSING", fmt.Sprintf("agent transform %q should declare an explicit policy", transform.Name), transform.File, transform.Name)
	}
	for _, evalName := range transform.Evals {
		if _, ok := evals[evalName]; !ok {
			r.addError("EVAL_REF_UNRESOLVED", fmt.Sprintf("transform %q references missing eval %q", transform.Name, evalName), transform.File, transform.Name)
		}
	}
}

func (r *Result) validateInput(transform Transform, input Input, sourceIDs map[string]SourceArtifact, artifactDefs map[string]Artifact) {
	if (input.Source == "") == (input.Ref == "") {
		r.addError("INPUT_REFERENCE_INVALID", fmt.Sprintf("transform %q input must declare exactly one of source or ref", transform.Name), transform.File, transform.Name)
		return
	}
	if input.Source != "" {
		if !validSourceRef(input.Source) {
			r.addError("SOURCE_REF_INVALID", fmt.Sprintf("source reference %q must use source_name.artifact_name", input.Source), transform.File, transform.Name)
			return
		}
		if _, ok := sourceIDs[input.Source]; !ok {
			r.addError("SOURCE_REF_UNRESOLVED", fmt.Sprintf("transform %q references missing source %q", transform.Name, input.Source), transform.File, transform.Name)
		}
		return
	}
	if !resourceNamePattern.MatchString(input.Ref) {
		r.addError("REF_INVALID", fmt.Sprintf("ref %q is not a valid artifact name", input.Ref), transform.File, transform.Name)
		return
	}
	if _, ok := artifactDefs[input.Ref]; !ok {
		r.addError("REF_UNRESOLVED", fmt.Sprintf("transform %q references missing artifact %q", transform.Name, input.Ref), transform.File, transform.Name)
	}
}

func (r *Result) validateName(code, kind, name, file string) {
	if name == "" {
		return
	}
	if !resourceNamePattern.MatchString(name) {
		r.addError(code, fmt.Sprintf("%s name %q must match %s", kind, name, resourceNamePattern.String()), file, name)
	}
}

func (r *Result) validateArtifactType(alias, file, resource string) {
	if alias == "" {
		r.addError("ARTIFACT_TYPE_REQUIRED", fmt.Sprintf("%s must declare artifact type", resource), file, resource)
		return
	}
	if err := config.ValidateArtifactType(alias); err != nil {
		r.addError("ARTIFACT_TYPE_UNSUPPORTED", err.Error(), file, resource)
	}
}

func (r *Result) validateArtifactPath(path, file, resource string) {
	if path == "" {
		r.addError("ARTIFACT_PATH_REQUIRED", fmt.Sprintf("artifact %q must declare path", resource), file, resource)
		return
	}
	clean, err := cleanProjectRelativePath(path)
	if err != nil {
		r.addError("ARTIFACT_PATH_INVALID", fmt.Sprintf("invalid artifact path %q: %v", path, err), file, resource)
		return
	}
	artifactRoot, err := projectAbs(r.ProjectDir, r.Config.ArtifactPath)
	if err != nil {
		r.addError("ARTIFACT_PATH_INVALID", err.Error(), file, resource)
		return
	}
	outputAbs := filepath.Join(r.ProjectDir, clean)
	if !isWithin(artifactRoot, outputAbs) {
		r.addError("PATH_OUTSIDE_ARTIFACT_PATH", fmt.Sprintf("artifact path %q must stay under %q", path, r.Config.ArtifactPath), file, resource)
	}
}

func (r *Result) validateSourcePath(path, file, resource string) {
	if path == "" {
		r.addError("SOURCE_PATH_REQUIRED", fmt.Sprintf("source %q must declare path", resource), file, resource)
		return
	}
	if isRemoteURI(path) {
		return
	}
	clean, err := cleanProjectRelativePath(path)
	if err != nil {
		r.addError("SOURCE_PATH_INVALID", fmt.Sprintf("invalid source path %q: %v", path, err), file, resource)
		return
	}
	sourceAbs := filepath.Join(r.ProjectDir, clean)
	artifactRoot, err := projectAbs(r.ProjectDir, r.Config.ArtifactPath)
	if err == nil && isWithin(artifactRoot, sourceAbs) {
		r.addError("SOURCE_PATH_IN_ARTIFACT_PATH", fmt.Sprintf("source path %q must not be under artifact_path %q", path, r.Config.ArtifactPath), file, resource)
	}
	if hasGlob(clean) {
		matches, err := filepath.Glob(filepath.Join(r.ProjectDir, clean))
		if err != nil {
			r.addError("SOURCE_GLOB_INVALID", fmt.Sprintf("invalid source glob %q: %v", path, err), file, resource)
			return
		}
		if len(matches) == 0 {
			r.addError("SOURCE_PATH_MISSING", fmt.Sprintf("source glob %q matched no files", path), file, resource)
		}
		return
	}
	if _, err := os.Stat(sourceAbs); err != nil {
		r.addError("SOURCE_PATH_MISSING", fmt.Sprintf("source path %q is not accessible: %v", path, err), file, resource)
	}
}

func (r *Result) validateExistingProjectPath(code, path, file, resource string) {
	clean, err := cleanProjectRelativePath(path)
	if err != nil {
		r.addError(code, fmt.Sprintf("invalid path %q: %v", path, err), file, resource)
		return
	}
	if _, err := os.Stat(filepath.Join(r.ProjectDir, clean)); err != nil {
		r.addError(code, fmt.Sprintf("path %q is not accessible: %v", path, err), file, resource)
	}
}

func (r *Result) validateProjectPath(code, path, file, resource string) {
	if _, err := cleanProjectRelativePath(path); err != nil {
		r.addError(code, fmt.Sprintf("invalid project path %q: %v", path, err), file, resource)
	}
}

func (r *Result) addError(code, message, file, resource string) {
	r.Diagnostics = append(r.Diagnostics, r.newDiagnostic(SeverityError, code, message, file, resource))
}

func (r *Result) addWarning(code, message, file, resource string) {
	r.Diagnostics = append(r.Diagnostics, r.newDiagnostic(SeverityWarning, code, message, file, resource))
}

func (r *Result) addUnsupportedField(file string, field unsupportedField) {
	r.Diagnostics = append(r.Diagnostics, Diagnostic{
		Severity: SeverityError,
		Code:     field.code,
		Message:  field.message,
		File:     file,
		Line:     field.line,
		Resource: field.resource,
		Hint:     diagnosticHint(field.code),
	})
}

func (r *Result) newDiagnostic(severity Severity, code, message, file, resource string) Diagnostic {
	return Diagnostic{
		Severity: severity,
		Code:     code,
		Message:  message,
		File:     file,
		Line:     r.diagnosticLine(file, resource),
		Resource: resource,
		Hint:     diagnosticHint(code),
	}
}

func (r *Result) diagnosticLine(file, resource string) int {
	if r.lineIndex == nil || file == "" || resource == "" {
		return 0
	}
	if line, ok := r.lineIndex[diagKey(file, resource)]; ok {
		return line
	}
	return 0
}

func diagKey(file, resource string) string {
	return file + "\x00" + resource
}

func diagnosticHint(code string) string {
	switch code {
	case "CONFIG_VERSION_MISSING":
		return "Add `config_version: 1` to fs_project.yml."
	case "REVIEW_UNSUPPORTED":
		return "Remove the review field. Use Git, PRs, CI, release tooling, or your publishing workflow for human approval."
	case "PROJECT_NAME_REQUIRED":
		return "Add a non-empty `name` to fs_project.yml."
	case "PROJECT_NAME_INVALID", "SOURCE_NAME_INVALID", "SOURCE_ARTIFACT_NAME_INVALID", "ARTIFACT_NAME_INVALID", "ASSET_NAME_INVALID", "POLICY_NAME_INVALID", "EVAL_NAME_INVALID", "TRANSFORM_NAME_INVALID":
		return "Use a name that starts with a letter and contains only letters, numbers, and underscores."
	case "ARTIFACT_TYPE_REQUIRED", "ARTIFACT_TYPE_UNSUPPORTED":
		return "Use a supported artifact type from docs/schema-and-versioning-spec.md."
	case "ARTIFACT_PATH_REQUIRED":
		return "Add `path:` under the artifact or transform output."
	case "PATH_OUTSIDE_ARTIFACT_PATH":
		return "Move the output under `artifact_path` or update artifact_path in fs_project.yml."
	case "SOURCE_PATH_MISSING":
		return "Create the source file/directory or update the source `path:`."
	case "ASSET_PATH_MISSING":
		return "Create the asset file or update the asset `path:`."
	case "REF_UNRESOLVED":
		return "Build or declare an upstream artifact with this name, or correct the `ref:` value."
	case "SOURCE_REF_UNRESOLVED", "SOURCE_REF_INVALID":
		return "Use `source: source_name.artifact_name` and ensure the source artifact exists."
	case "ASSET_REF_UNRESOLVED":
		return "Declare the asset in an assets YAML file or correct the `ref:` value."
	case "EVAL_REF_UNRESOLVED":
		return "Declare the eval in an evals YAML file or remove it from the transform."
	case "POLICY_REF_UNRESOLVED":
		return "Declare the policy in a policies YAML file or correct the policy name."
	case "TRANSFORM_RUNNER_REQUIRED", "RUNNER_UNDECLARED":
		return "Declare the runner in fs_project.yml or install a runner discoverable by fbt."
	case "TRANSFORM_INPUTS_REQUIRED":
		return "Add at least one `inputs:` entry using `source:` or `ref:`."
	case "TRANSFORM_OUTPUTS_REQUIRED":
		return "Add at least one `outputs:` entry."
	case "INPUT_REFERENCE_INVALID":
		return "Each input must declare exactly one of `source:` or `ref:`."
	default:
		return ""
	}
}

func (r Result) hasErrors() bool {
	return HasErrors(r.Diagnostics)
}

func HasErrors(diagnostics []Diagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == SeverityError {
			return true
		}
	}
	return false
}

func cleanProjectRelativePath(path string) (string, error) {
	return security.CleanProjectRelative(path)
}

func projectAbs(root, path string) (string, error) {
	clean, err := cleanProjectRelativePath(path)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, clean), nil
}

func cleanPath(path string) string {
	clean, err := cleanProjectRelativePath(path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(clean)
}

func isWithin(parent, child string) bool {
	return security.IsWithin(parent, child)
}

func isRemoteURI(path string) bool {
	return strings.Contains(path, "://")
}

func hasGlob(path string) bool {
	return strings.ContainsAny(path, "*?[")
}

func validSourceRef(value string) bool {
	parts := strings.Split(value, ".")
	return len(parts) == 2 && resourceNamePattern.MatchString(parts[0]) && resourceNamePattern.MatchString(parts[1])
}
