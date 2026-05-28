package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAllPluginManifests(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "openai", ManifestFileName), `name: fbt-openai
version: 0.1.0
protocol: stdio_jsonrpc
command: fbt-openai-runner
provides:
  - runner: openai.responses
    type: llm
`)
	manifests, err := LoadAll(root)
	if err != nil {
		t.Fatalf("load manifests: %v", err)
	}
	if len(manifests) != 1 {
		t.Fatalf("expected one manifest, got %d", len(manifests))
	}
	if manifests[0].Provides[0].Runner != "openai.responses" {
		t.Fatalf("unexpected runner: %q", manifests[0].Provides[0].Runner)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
