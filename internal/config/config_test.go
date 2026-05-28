package config

import "testing"

func TestDecodeProjectFileRequiresConfigVersion(t *testing.T) {
	projectFile, err := DecodeProjectFile([]byte("name: demo\n"))
	if err != nil {
		t.Fatalf("decode project file: %v", err)
	}
	if projectFile.VersionStatus != VersionMissing {
		t.Fatalf("expected missing version status, got %v", projectFile.VersionStatus)
	}
}

func TestDecodeProjectFileAppliesDefaults(t *testing.T) {
	projectFile, err := DecodeProjectFile([]byte("name: demo\nconfig_version: 1\n"))
	if err != nil {
		t.Fatalf("decode project file: %v", err)
	}
	cfg := projectFile.Config
	if cfg.ArtifactPath != "target/artifacts" {
		t.Fatalf("unexpected artifact path default: %q", cfg.ArtifactPath)
	}
	if cfg.State.Path != ".fbt/state" {
		t.Fatalf("unexpected state path default: %q", cfg.State.Path)
	}
	if got := cfg.SourcePaths[0]; got != "sources" {
		t.Fatalf("unexpected source path default: %q", got)
	}
}

func TestArtifactTypeRegistry(t *testing.T) {
	artifactType, ok := LookupArtifactType("markdown_directory")
	if !ok {
		t.Fatal("expected markdown_directory to be registered")
	}
	if artifactType.PathKind != PathKindDirectory {
		t.Fatalf("unexpected path kind: %q", artifactType.PathKind)
	}
	if err := ValidateArtifactType("nope"); err == nil {
		t.Fatal("expected unsupported artifact type to fail")
	}
	if err := ValidateArtifactType("x.example.report.v1"); err != nil {
		t.Fatalf("expected custom artifact type to pass: %v", err)
	}
}
