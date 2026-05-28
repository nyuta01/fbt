package templates

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nyuta01/fbt/internal/parser"
)

func TestCreateSupportProjectParses(t *testing.T) {
	root := filepath.Join(t.TempDir(), "knowledge_ops")
	result, err := CreateProject(Options{ProjectName: "knowledge_ops", Destination: root, Template: "support"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if len(result.Files) == 0 {
		t.Fatal("expected generated files")
	}
	if _, err := parser.ParseProject(parser.Options{ProjectDir: root}); err != nil {
		t.Fatalf("parse generated project: %v", err)
	}
}

func TestCreateIncidentProjectParses(t *testing.T) {
	root := filepath.Join(t.TempDir(), "incident_ops")
	if _, err := CreateProject(Options{ProjectName: "incident_ops", Destination: root, Template: "incident"}); err != nil {
		t.Fatalf("create project: %v", err)
	}
	if _, err := parser.ParseProject(parser.Options{ProjectDir: root}); err != nil {
		t.Fatalf("parse generated project: %v", err)
	}
}

func TestCreateProjectRefusesOverwrite(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "fs_project.yml"), []byte("existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateProject(Options{ProjectName: "demo", Destination: root, Template: "blank"}); err == nil {
		t.Fatal("expected overwrite refusal")
	}
	if _, err := CreateProject(Options{ProjectName: "demo", Destination: root, Template: "blank", Force: true}); err != nil {
		t.Fatalf("force should overwrite: %v", err)
	}
}
