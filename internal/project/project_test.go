package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverFindsParentProject(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ConfigFileName), []byte("name: demo\nconfig_version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	project, err := Discover(nested)
	if err != nil {
		t.Fatalf("discover project: %v", err)
	}
	if project.RootDir != root {
		t.Fatalf("expected root %q, got %q", root, project.RootDir)
	}
}
