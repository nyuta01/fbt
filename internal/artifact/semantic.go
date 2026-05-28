package artifact

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nyuta01/fbt/internal/config"
	"github.com/nyuta01/fbt/internal/security"
)

type markdownHeading struct {
	Level int    `json:"level"`
	Text  string `json:"text"`
}

type markdownStructure struct {
	Headings       []markdownHeading `json:"headings"`
	CodeBlockCount int               `json:"code_block_count"`
}

func SemanticDescriptor(root, path, artifactAlias string) (map[string]any, error) {
	artifactType, ok := config.LookupArtifactType(artifactAlias)
	if !ok {
		return nil, nil
	}
	switch artifactAlias {
	case "text", "markdown", "markdown_directory":
	default:
		return nil, nil
	}

	content, err := readSemanticText(root, path, artifactType.PathKind)
	if err != nil {
		return nil, err
	}
	if content == "" {
		return nil, nil
	}

	normalized := normalizeText(content)
	descriptor := map[string]any{
		"text_normalized_v1": map[string]any{
			"digest":     digestString(normalized),
			"char_count": len([]rune(normalized)),
			"word_count": len(strings.Fields(normalized)),
			"line_count": lineCount(normalized),
		},
	}
	if artifactAlias == "markdown" || artifactAlias == "markdown_directory" {
		structure := markdownAST(content)
		descriptor["markdown_ast_v1"] = map[string]any{
			"digest":           digestCanonical(structure),
			"heading_count":    len(structure.Headings),
			"code_block_count": structure.CodeBlockCount,
			"headings":         structure.Headings,
		}
	}
	return descriptor, nil
}

func readSemanticText(root, path string, kind config.ArtifactPathKind) (string, error) {
	abs, err := security.ResolveProjectRelative(root, path)
	if err != nil {
		return "", err
	}
	if err := security.RejectSymlinkPath(root, abs); err != nil {
		return "", err
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", errSymlink(path)
	}
	if kind == config.PathKindFile {
		data, err := os.ReadFile(abs)
		return string(data), err
	}

	var files []string
	if err := filepath.WalkDir(abs, func(entryPath string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entryPath == abs {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return errSymlink(entryPath)
		}
		if d.IsDir() {
			return nil
		}
		if semanticTextFile(entryPath) {
			files = append(files, entryPath)
		}
		return nil
	}); err != nil {
		return "", err
	}
	sort.Strings(files)
	var builder strings.Builder
	for _, file := range files {
		rel, err := filepath.Rel(abs, file)
		if err != nil {
			return "", err
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return "", err
		}
		builder.WriteString("\n--- ")
		builder.WriteString(filepath.ToSlash(rel))
		builder.WriteString(" ---\n")
		builder.Write(data)
		if !strings.HasSuffix(builder.String(), "\n") {
			builder.WriteByte('\n')
		}
	}
	return builder.String(), nil
}

func semanticTextFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown", ".txt", ".html", ".htm":
		return true
	default:
		return false
	}
}

func normalizeText(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	var lines []string
	for _, line := range strings.Split(content, "\n") {
		normalized := strings.Join(strings.Fields(line), " ")
		if normalized == "" {
			continue
		}
		lines = append(lines, normalized)
	}
	return strings.Join(lines, "\n")
}

func markdownAST(content string) markdownStructure {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	structure := markdownStructure{}
	inFence := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			if inFence {
				structure.CodeBlockCount++
			}
			continue
		}
		if inFence || !strings.HasPrefix(trimmed, "#") {
			continue
		}
		level := 0
		for level < len(trimmed) && level < 6 && trimmed[level] == '#' {
			level++
		}
		if level == 0 || level >= len(trimmed) || trimmed[level] != ' ' {
			continue
		}
		text := strings.TrimSpace(trimmed[level:])
		text = strings.TrimSpace(strings.TrimRight(text, "#"))
		if text == "" {
			text = "(unnamed)"
		}
		structure.Headings = append(structure.Headings, markdownHeading{Level: level, Text: text})
	}
	return structure
}

func digestString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func digestCanonical(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return digestString("")
	}
	return digestString(string(data))
}

func lineCount(value string) int {
	if value == "" {
		return 0
	}
	return len(strings.Split(value, "\n"))
}

func errSymlink(path string) error {
	return &semanticError{message: "symlink rejected: " + path}
}

type semanticError struct {
	message string
}

func (e *semanticError) Error() string {
	return e.message
}
