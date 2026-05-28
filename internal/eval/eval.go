package eval

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nyuta01/fbt/internal/manifest"
	"github.com/nyuta01/fbt/internal/state"
)

type Outcome struct {
	Results    []state.EvaluationResult
	Confidence string
}

func RunForCandidate(root string, transform manifest.TransformResource, evals map[string]manifest.EvalResource, artifactVersionID, transformRunID, candidatePath string) (Outcome, error) {
	var outcome Outcome
	var firstErr error
	for i, evalID := range transform.Evals {
		evalResource, ok := evals[evalID]
		if !ok {
			err := fmt.Errorf("eval resource not found: %s", evalID)
			if firstErr == nil {
				firstErr = err
			}
			outcome.Results = append(outcome.Results, state.EvaluationResult{
				ResultID:          resultID(evalID, i),
				EvalID:            evalID,
				ArtifactVersionID: artifactVersionID,
				TransformRunID:    transformRunID,
				Status:            "error",
			})
			continue
		}

		result := state.EvaluationResult{
			ResultID:          resultID(evalResource.UniqueID, i),
			EvalID:            evalResource.UniqueID,
			ArtifactVersionID: artifactVersionID,
			TransformRunID:    transformRunID,
			Status:            "skipped",
			Runner:            evalResource.Runner,
		}
		switch evalResource.EvalType {
		case "deterministic":
			status, score, threshold, err := runDeterministic(root, candidatePath, evalResource.Config)
			result.Status = status
			result.Score = score
			result.Threshold = threshold
			if err != nil && firstErr == nil {
				firstErr = fmt.Errorf("%s: %w", evalResource.UniqueID, err)
			}
			if status == "fail" && firstErr == nil {
				firstErr = fmt.Errorf("eval failed: %s", evalResource.UniqueID)
			}
			if status == "pass" && evalResource.GrantsConfidence != "" {
				result.GrantsConfidence = evalResource.GrantsConfidence
				outcome.Confidence = maxConfidence(outcome.Confidence, evalResource.GrantsConfidence)
			}
		case "semantic", "llm_judge":
			result.Status = "skipped"
		default:
			result.Status = "error"
			if firstErr == nil {
				firstErr = fmt.Errorf("unsupported eval type %q for %s", evalResource.EvalType, evalResource.UniqueID)
			}
		}
		outcome.Results = append(outcome.Results, result)
	}
	return outcome, firstErr
}

func runDeterministic(root, candidatePath string, config map[string]any) (string, *float64, *float64, error) {
	content, err := readCandidateText(root, candidatePath)
	if err != nil {
		return "error", nil, nil, err
	}

	var checks []bool
	sections := stringSlice(config["sections"])
	for _, section := range sections {
		checks = append(checks, hasSection(content, section))
	}
	contains := stringSlice(firstPresent(config, "contains", "required_text", "must_contain"))
	for _, value := range contains {
		checks = append(checks, strings.Contains(content, value))
	}
	if len(checks) == 0 {
		nonEmpty := true
		if value, ok := config["non_empty"].(bool); ok {
			nonEmpty = value
		}
		if nonEmpty {
			checks = append(checks, strings.TrimSpace(content) != "")
		}
	}

	passed := 0
	for _, check := range checks {
		if check {
			passed++
		}
	}
	score := 1.0
	threshold := 1.0
	if len(checks) > 0 {
		score = float64(passed) / float64(len(checks))
	}
	if passed == len(checks) {
		return "pass", &score, &threshold, nil
	}
	return "fail", &score, &threshold, nil
}

func readCandidateText(root, candidatePath string) (string, error) {
	path := candidatePath
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, candidatePath)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("symlink rejected: %s", candidatePath)
	}
	if !info.IsDir() {
		data, err := os.ReadFile(path)
		return string(data), err
	}

	var files []string
	if err := filepath.WalkDir(path, func(entryPath string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entryPath == path {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink rejected: %s", entryPath)
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, entryPath)
		return nil
	}); err != nil {
		return "", err
	}
	sort.Strings(files)
	var builder strings.Builder
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", err
		}
		builder.Write(data)
		if !strings.HasSuffix(builder.String(), "\n") {
			builder.WriteByte('\n')
		}
	}
	return builder.String(), nil
}

func hasSection(content, section string) bool {
	target := strings.TrimSpace(section)
	if target == "" {
		return true
	}
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			trimmed = strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
		}
		if strings.EqualFold(trimmed, target) {
			return true
		}
	}
	return strings.Contains(content, section)
}

func stringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				values = append(values, text)
			}
		}
		return values
	case string:
		return []string{typed}
	default:
		return nil
	}
}

func firstPresent(config map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := config[key]; ok {
			return value
		}
	}
	return nil
}

func resultID(evalID string, sequence int) string {
	name := strings.TrimPrefix(evalID, "eval.")
	return fmt.Sprintf("evaluation_result.%s.%d.%d", name, time.Now().UTC().UnixNano(), sequence)
}

func maxConfidence(left, right string) string {
	order := map[string]int{
		"experimental": 0,
		"structural":   1,
		"semantic":     2,
		"exact":        3,
	}
	if left == "" {
		return right
	}
	if right == "" {
		return left
	}
	if order[right] > order[left] {
		return right
	}
	return left
}
