package outputs

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/nyuta01/fbt/sdk/go/protocol"
)

func IsDirectoryType(artifactType string) bool {
	return artifactType == "directory" || strings.HasSuffix(artifactType, "_directory")
}

func Collect(root string) ([]map[string]any, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	candidates := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		candidates = append(candidates, map[string]any{
			"name": entry.Name(),
			"path": filepath.Join(root, entry.Name()),
		})
	}
	return candidates, nil
}

func WriteText(root string, output protocol.DeclaredOutput, content []byte) (map[string]any, error) {
	name := output.Name
	if name == "" {
		name = "output"
	}
	artifactType := output.ArtifactType
	if artifactType == "" {
		artifactType = "markdown"
	}
	path := filepath.Join(root, name)
	if IsDirectoryType(artifactType) {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(path, "index.md"), content, 0o644); err != nil {
			return nil, err
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			return nil, err
		}
	}
	return map[string]any{
		"name":          name,
		"artifact_type": artifactType,
		"path":          path,
		"declared_path": output.DeclaredPath,
	}, nil
}
