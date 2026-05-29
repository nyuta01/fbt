package outputs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nyuta01/fbt/sdk/go/protocol"
)

func TestWriteTextFileAndCollect(t *testing.T) {
	root := t.TempDir()
	candidate, err := WriteText(root, protocol.DeclaredOutput{
		Name:         "manual.md",
		ArtifactType: "markdown",
	}, []byte("# Manual\n"))
	if err != nil {
		t.Fatal(err)
	}
	if candidate["name"] != "manual.md" {
		t.Fatalf("candidate = %+v", candidate)
	}
	if _, err := os.Stat(filepath.Join(root, "manual.md")); err != nil {
		t.Fatal(err)
	}
	candidates, err := Collect(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0]["name"] != "manual.md" {
		t.Fatalf("candidates = %+v", candidates)
	}
}

func TestWriteTextDirectory(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteText(root, protocol.DeclaredOutput{
		Name:         "manual",
		ArtifactType: "markdown_directory",
	}, []byte("# Manual\n")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "manual", "index.md")); err != nil {
		t.Fatal(err)
	}
}
