package runner

import (
	"path/filepath"
	"testing"

	"github.com/nyuta01/fbt/internal/config"
	"github.com/nyuta01/fbt/internal/protocol"
)

func TestReadLockfileAndValidateMatch(t *testing.T) {
	root := t.TempDir()
	command := writeExecutable(t, root, "bin/runner")
	writeFile(t, root, LockfileName, `{
  "fbt_schema_version": "https://schemas.fbt.dev/fbt/runner-lock/v1.json",
  "lockfile_version": 1,
  "runners": {
    "demo.llm": {
      "protocol_version": "0.1",
      "command": "runner",
      "checksums": {
        "command": "`+fileDigest(command)+`"
      },
      "capabilities": {
        "transform_types": ["llm"],
        "artifact_types": ["markdown_directory"],
        "output_candidates": true
      }
    }
  }
}`)

	lock, ok, err := ReadLockfile(root)
	if err != nil || !ok {
		t.Fatalf("read lockfile: ok=%v err=%v", ok, err)
	}
	resolved := Resolved{Name: "demo.llm", Command: "bin/runner", CommandPath: command}
	if diagnostics := ValidateLockResolved(lock, resolved); HasErrors(diagnostics) {
		t.Fatalf("expected resolved lock match, got %+v", diagnostics)
	}
	initResult := protocol.InitializeResult{
		Protocol: map[string]any{"version": "0.1"},
		Capabilities: map[string]any{
			"transform_types":   []any{"llm"},
			"artifact_types":    []any{"markdown_directory"},
			"output_candidates": true,
		},
	}
	if diagnostics := ValidateLockInitialized(lock, "demo.llm", initResult); HasErrors(diagnostics) {
		t.Fatalf("expected initialized lock match, got %+v", diagnostics)
	}
}

func TestLockDiagnosticsReportMismatchAndCoverage(t *testing.T) {
	root := t.TempDir()
	command := writeExecutable(t, root, "bin/runner")
	lock := Lockfile{
		FBTSchemaVersion: LockfileSchemaVersion,
		LockfileVersion:  SupportedLockfileVersion,
		Runners: map[string]LockEntry{
			"demo.llm": {
				ProtocolVersion: "0.2",
				Command:         "other-runner",
				Checksums:       map[string]string{"command": "sha256:0000"},
				Capabilities: LockCapabilities{
					TransformTypes:   []string{"agent"},
					ArtifactTypes:    []string{"pdf"},
					OutputCandidates: boolPtr(true),
				},
			},
			"unused.runner": {ProtocolVersion: "0.1"},
		},
	}
	resolved := Resolved{Name: "demo.llm", Command: "bin/runner", CommandPath: command}
	if diagnostics := ValidateLockResolved(lock, resolved); !HasErrors(diagnostics) || !hasDiagnosticCode(diagnostics, "RUNNER_LOCK_MISMATCH") {
		t.Fatalf("expected resolved mismatch diagnostics, got %+v", diagnostics)
	}
	initResult := protocol.InitializeResult{
		Protocol: map[string]any{"version": "0.1"},
		Capabilities: map[string]any{
			"transform_types":   []any{"llm"},
			"artifact_types":    []any{"markdown_directory"},
			"output_candidates": false,
		},
	}
	if diagnostics := ValidateLockInitialized(lock, "demo.llm", initResult); !HasErrors(diagnostics) || !hasDiagnosticCode(diagnostics, "RUNNER_LOCK_MISMATCH") {
		t.Fatalf("expected initialized mismatch diagnostics, got %+v", diagnostics)
	}
	coverage := ValidateLockCoverage(lock, []Resolved{resolved, {Name: "missing.runner"}})
	if !lockDiagnosticsContain(coverage, "RUNNER_LOCK_UNUSED") || !lockDiagnosticsContain(coverage, "RUNNER_LOCK_MISSING") {
		t.Fatalf("expected unused and missing coverage diagnostics, got %+v", coverage)
	}
}

func TestReadLockfileRejectsUnsupportedShape(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, LockfileName, `{
  "fbt_schema_version": "https://schemas.fbt.dev/fbt/runner-lock/v2.json",
  "lockfile_version": 2,
  "runners": {}
}`)
	if _, ok, err := ReadLockfile(root); !ok || err == nil {
		t.Fatalf("expected unsupported lockfile error, ok=%v err=%v", ok, err)
	}

	writeFile(t, root, LockfileName, `{
  "fbt_schema_version": "https://schemas.fbt.dev/fbt/runner-lock/v1.json",
  "lockfile_version": 1,
  "runners": {
    "demo.llm": {
      "protocol_version": "0.1",
      "unexpected": true
    }
  }
}`)
	if _, ok, err := ReadLockfile(root); !ok || err == nil {
		t.Fatalf("expected strict unknown-field lockfile error, ok=%v err=%v", ok, err)
	}
}

func boolPtr(value bool) *bool {
	return &value
}

func lockDiagnosticsContain(diagnostics []LockDiagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Diagnostic.Code == code {
			return true
		}
	}
	return false
}

func TestPluginResolutionCarriesManifestDigest(t *testing.T) {
	root := t.TempDir()
	writeExecutable(t, root, "plugins/demo/runner")
	manifestPath := filepath.Join(root, "plugins/demo", "fbt_plugin.yml")
	writeFile(t, root, "plugins/demo/fbt_plugin.yml", `name: demo
version: 0.1.0
protocol: stdio_jsonrpc
command: ./runner
checksum:
  go_module: h1:test
provides:
  - runner: demo.llm
    type: llm
`)
	resolved, err := NewDiscovery(root, config.ProjectConfig{}).Resolve("demo.llm")
	if err != nil {
		t.Fatalf("resolve plugin: %v", err)
	}
	if resolved.ManifestDigest != fileDigest(manifestPath) {
		t.Fatalf("expected manifest digest, got %q", resolved.ManifestDigest)
	}
	if resolved.Checksums["go_module"] != "h1:test" {
		t.Fatalf("expected plugin checksums, got %+v", resolved.Checksums)
	}
}
