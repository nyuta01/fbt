package artifact

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestDescribeFileUsesExactBytes(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "report.md", "# Report\n")

	descriptor, err := Describe(root, "report.md", "markdown")
	if err != nil {
		t.Fatalf("describe file: %v", err)
	}
	sum := sha256.Sum256([]byte("# Report\n"))
	want := "sha256:" + hex.EncodeToString(sum[:])
	if descriptor.Digest != want {
		t.Fatalf("digest mismatch: got %s want %s", descriptor.Digest, want)
	}
	if descriptor.Size == nil || *descriptor.Size != int64(len("# Report\n")) {
		t.Fatalf("unexpected size: %+v", descriptor.Size)
	}
	if descriptor.ArtifactType != "fbt.artifact.markdown_document.v1" {
		t.Fatalf("unexpected artifact type: %s", descriptor.ArtifactType)
	}
}

func TestDescribeDirectoryCanonicalizesContent(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()
	writeFile(t, first, "out/a/one.md", "one\n")
	writeFile(t, first, "out/b/two.md", "two\n")
	writeFile(t, second, "out/b/two.md", "two\n")
	writeFile(t, second, "out/a/one.md", "one\n")

	firstDescriptor, err := Describe(first, "out", "markdown_directory")
	if err != nil {
		t.Fatalf("describe first dir: %v", err)
	}
	secondDescriptor, err := Describe(second, "out", "markdown_directory")
	if err != nil {
		t.Fatalf("describe second dir: %v", err)
	}
	if firstDescriptor.Digest != secondDescriptor.Digest {
		t.Fatalf("directory digest should be stable: %s != %s", firstDescriptor.Digest, secondDescriptor.Digest)
	}
	if firstDescriptor.FileCount != 2 {
		t.Fatalf("unexpected file count: %d", firstDescriptor.FileCount)
	}
	if firstDescriptor.Size != nil {
		t.Fatalf("directory size should be null, got %+v", firstDescriptor.Size)
	}
}

func TestDescribeDirectoryRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "safe/file.md", "safe\n")
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "safe", "link.md")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := Describe(root, "safe", "markdown_directory"); err == nil {
		t.Fatal("expected symlink to be rejected")
	}
}

func TestDescribeRejectsPathEscape(t *testing.T) {
	root := t.TempDir()
	if _, err := Describe(root, "../outside.md", "markdown"); err == nil {
		t.Fatal("expected path escape to be rejected")
	}
}

func TestSemanticDescriptorForTextNormalizesWhitespace(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "first.txt", "Hello   world\r\n\nAgain\n")
	writeFile(t, root, "second.txt", "Hello world\nAgain\n")

	first, err := SemanticDescriptor(root, "first.txt", "text")
	if err != nil {
		t.Fatalf("semantic first: %v", err)
	}
	second, err := SemanticDescriptor(root, "second.txt", "text")
	if err != nil {
		t.Fatalf("semantic second: %v", err)
	}
	if first["text_normalized_v1"].(map[string]any)["digest"] != second["text_normalized_v1"].(map[string]any)["digest"] {
		t.Fatalf("normalized text digest should ignore whitespace: %+v != %+v", first, second)
	}
}

func TestSemanticDescriptorForMarkdownStructure(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "report.md", "# Title\n\nBody\n\n## Details\n\n```go\n# not heading\n```\n")

	semantic, err := SemanticDescriptor(root, "report.md", "markdown")
	if err != nil {
		t.Fatalf("semantic markdown: %v", err)
	}
	if semantic["text_normalized_v1"] == nil || semantic["markdown_ast_v1"] == nil {
		t.Fatalf("expected text and markdown descriptors: %+v", semantic)
	}
	markdown := semantic["markdown_ast_v1"].(map[string]any)
	if markdown["heading_count"] != 2 || markdown["code_block_count"] != 1 {
		t.Fatalf("unexpected markdown structure: %+v", markdown)
	}
}

func TestVersionID(t *testing.T) {
	digest := "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	id, err := VersionID("knowledge_ops", "weekly_report", digest)
	if err != nil {
		t.Fatalf("version id: %v", err)
	}
	want := "artifact_version.knowledge_ops.weekly_report.sha256_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if id != want {
		t.Fatalf("unexpected version id: %s", id)
	}
	if _, err := VersionID("knowledge_ops", "weekly_report", "sha256:ABC"); err == nil {
		t.Fatal("expected invalid digest to fail")
	}
}

func writeFile(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
