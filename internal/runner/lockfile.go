package runner

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/nyuta01/fbt/internal/protocol"
)

const (
	LockfileName                   = "fbt.lock.json"
	LockfileSchemaVersion          = "https://schemas.fbt.dev/fbt/runner-lock/v1.json"
	SupportedLockfileVersion       = 1
	comparableCommandChecksumKey   = "command"
	comparableBinaryChecksumKey    = "binary"
	comparablePlatformChecksumKey  = runtime.GOOS + "-" + runtime.GOARCH
	comparablePluginManifestDigest = "manifest"
)

var ErrLockIncompatible = errors.New("runner lock incompatible")

type Lockfile struct {
	FBTSchemaVersion string               `json:"fbt_schema_version"`
	LockfileVersion  int                  `json:"lockfile_version"`
	Runners          map[string]LockEntry `json:"runners"`
	Meta             map[string]any       `json:"meta,omitempty"`
}

type LockEntry struct {
	Source          string            `json:"source,omitempty"`
	Version         string            `json:"version,omitempty"`
	ProtocolVersion string            `json:"protocol_version"`
	Command         string            `json:"command,omitempty"`
	ManifestDigest  string            `json:"manifest_digest,omitempty"`
	Checksums       map[string]string `json:"checksums,omitempty"`
	Capabilities    LockCapabilities  `json:"capabilities,omitempty"`
	Meta            map[string]any    `json:"meta,omitempty"`
}

type LockCapabilities struct {
	TransformTypes   []string `json:"transform_types,omitempty"`
	ArtifactTypes    []string `json:"artifact_types,omitempty"`
	OutputCandidates *bool    `json:"output_candidates,omitempty"`
}

type LockDiagnostic struct {
	RunnerName string
	Diagnostic Diagnostic
}

func ReadLockfile(projectDir string) (Lockfile, bool, error) {
	path := filepath.Join(projectDir, LockfileName)
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return Lockfile{}, false, nil
	}
	if err != nil {
		return Lockfile{}, true, err
	}
	defer file.Close()

	var lock Lockfile
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&lock); err != nil {
		return Lockfile{}, true, fmt.Errorf("malformed %s: %w", LockfileName, err)
	}
	if lock.FBTSchemaVersion != LockfileSchemaVersion {
		return Lockfile{}, true, fmt.Errorf("unsupported %s schema %q", LockfileName, lock.FBTSchemaVersion)
	}
	if lock.LockfileVersion != SupportedLockfileVersion {
		return Lockfile{}, true, fmt.Errorf("unsupported %s version %d", LockfileName, lock.LockfileVersion)
	}
	if lock.Runners == nil {
		return Lockfile{}, true, fmt.Errorf("%s runners must be present", LockfileName)
	}
	for name, entry := range lock.Runners {
		if strings.TrimSpace(name) == "" {
			return Lockfile{}, true, fmt.Errorf("%s runner names must not be empty", LockfileName)
		}
		if entry.ProtocolVersion == "" {
			return Lockfile{}, true, fmt.Errorf("%s runner %s protocol_version is required", LockfileName, name)
		}
	}
	return lock, true, nil
}

func (l Lockfile) EntryDigest(name string) string {
	entry, ok := l.Runners[name]
	if !ok {
		return ""
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func ValidateLockCoverage(lock Lockfile, resolved []Resolved) []LockDiagnostic {
	if len(lock.Runners) == 0 {
		return nil
	}
	resolvedNames := map[string]struct{}{}
	var diagnostics []LockDiagnostic
	for _, runner := range resolved {
		resolvedNames[runner.Name] = struct{}{}
		if _, ok := lock.Runners[runner.Name]; !ok {
			diagnostics = append(diagnostics, LockDiagnostic{
				RunnerName: runner.Name,
				Diagnostic: Diagnostic{
					Severity: "warning",
					Code:     "RUNNER_LOCK_MISSING",
					Message:  fmt.Sprintf("runner %s has no %s entry", runner.Name, LockfileName),
				},
			})
		}
	}
	diagnostics = append(diagnostics, lockUnusedDiagnostics(lock, resolvedNames)...)
	return diagnostics
}

func lockUnusedDiagnostics(lock Lockfile, resolvedNames map[string]struct{}) []LockDiagnostic {
	var names []string
	for name := range lock.Runners {
		if _, ok := resolvedNames[name]; !ok {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	diagnostics := make([]LockDiagnostic, 0, len(names))
	for _, name := range names {
		diagnostics = append(diagnostics, LockDiagnostic{
			RunnerName: name,
			Diagnostic: Diagnostic{
				Severity: "warning",
				Code:     "RUNNER_LOCK_UNUSED",
				Message:  fmt.Sprintf("%s entry has no matching configured runner: %s", LockfileName, name),
			},
		})
	}
	return diagnostics
}

func ValidateLockResolved(lock Lockfile, resolved Resolved) []Diagnostic {
	entry, ok := lock.Runners[resolved.Name]
	if !ok {
		return nil
	}
	var diagnostics []Diagnostic
	if entry.Command != "" && !lockCommandMatches(entry.Command, resolved) {
		diagnostics = append(diagnostics, lockMismatch("command", entry.Command, resolved.Command))
	}
	if entry.Version != "" && resolved.Version != "" && entry.Version != resolved.Version {
		diagnostics = append(diagnostics, lockMismatch("version", entry.Version, resolved.Version))
	}
	if entry.ManifestDigest != "" && entry.ManifestDigest != resolved.ManifestDigest {
		diagnostics = append(diagnostics, lockMismatch("manifest_digest", entry.ManifestDigest, emptyAsUnavailable(resolved.ManifestDigest)))
	}
	diagnostics = append(diagnostics, checksumDiagnostics(entry, resolved)...)
	return diagnostics
}

func ValidateLockInitialized(lock Lockfile, runnerName string, result protocol.InitializeResult) []Diagnostic {
	entry, ok := lock.Runners[runnerName]
	if !ok {
		return nil
	}
	var diagnostics []Diagnostic
	version, _ := result.Protocol["version"].(string)
	if entry.ProtocolVersion != "" && entry.ProtocolVersion != version {
		diagnostics = append(diagnostics, lockMismatch("protocol_version", entry.ProtocolVersion, emptyAsUnavailable(version)))
	}
	capabilities := entry.Capabilities
	actualTransformTypes := stringSet(result.Capabilities["transform_types"])
	for _, expected := range capabilities.TransformTypes {
		if !containsCapability(actualTransformTypes, expected) {
			diagnostics = append(diagnostics, lockMismatch("capabilities.transform_types", expected, "missing"))
		}
	}
	actualArtifactTypes := stringSet(result.Capabilities["artifact_types"])
	for _, expected := range capabilities.ArtifactTypes {
		if !containsCapability(actualArtifactTypes, expected) {
			diagnostics = append(diagnostics, lockMismatch("capabilities.artifact_types", expected, "missing"))
		}
	}
	if capabilities.OutputCandidates != nil {
		actual := boolCapability(result.Capabilities, "output_candidates")
		if actual != *capabilities.OutputCandidates {
			diagnostics = append(diagnostics, lockMismatch("capabilities.output_candidates", fmt.Sprintf("%v", *capabilities.OutputCandidates), fmt.Sprintf("%v", actual)))
		}
	}
	return diagnostics
}

func LockOKDiagnostic() Diagnostic {
	return Diagnostic{Severity: "info", Code: "RUNNER_LOCK_OK", Message: "runner lock entry matches resolved runner identity"}
}

func LockError(diagnostics []Diagnostic) error {
	var messages []string
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == "error" {
			messages = append(messages, diagnostic.Message)
		}
	}
	if len(messages) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrLockIncompatible, strings.Join(messages, "; "))
}

func lockMismatch(field, expected, actual string) Diagnostic {
	return Diagnostic{
		Severity: "error",
		Code:     "RUNNER_LOCK_MISMATCH",
		Message:  fmt.Sprintf("runner lock mismatch for %s: expected %s, got %s", field, expected, actual),
	}
}

func checksumDiagnostics(entry LockEntry, resolved Resolved) []Diagnostic {
	if len(entry.Checksums) == 0 {
		return nil
	}
	actual := comparableChecksums(resolved)
	var diagnostics []Diagnostic
	for key, expected := range entry.Checksums {
		if actualValue, ok := actual[key]; ok && actualValue != expected {
			diagnostics = append(diagnostics, lockMismatch("checksums."+key, expected, actualValue))
		}
	}
	return diagnostics
}

func comparableChecksums(resolved Resolved) map[string]string {
	out := copyStringMap(resolved.Checksums)
	if out == nil {
		out = map[string]string{}
	}
	if resolved.ManifestDigest != "" {
		out[comparablePluginManifestDigest] = resolved.ManifestDigest
	}
	if resolved.CommandPath != "" {
		if digest := fileDigest(resolved.CommandPath); digest != "" {
			out[comparableCommandChecksumKey] = digest
			out[comparableBinaryChecksumKey] = digest
			out[comparablePlatformChecksumKey] = digest
			out[filepath.Base(resolved.CommandPath)] = digest
		}
	}
	return out
}

func lockCommandMatches(expected string, resolved Resolved) bool {
	candidates := map[string]struct{}{}
	for _, value := range []string{
		resolved.Command,
		filepath.Base(resolved.Command),
		resolved.CommandPath,
		filepath.Base(resolved.CommandPath),
	} {
		if value != "" && value != "." {
			candidates[value] = struct{}{}
		}
	}
	_, ok := candidates[expected]
	return ok
}

func fileDigest(path string) string {
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func copyStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func emptyAsUnavailable(value string) string {
	if value == "" {
		return "unavailable"
	}
	return value
}
