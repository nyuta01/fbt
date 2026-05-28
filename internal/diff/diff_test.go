package diff

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompareTextReportsMarkdownSectionChanges(t *testing.T) {
	result := CompareText("# Summary\nold\n# Details\nsame\n", "# Summary\nnew\n# Added\nfresh\n# Details\nsame\n", "old", "new")
	if !strings.Contains(result.Unified, "-old") || !strings.Contains(result.Unified, "+new") {
		t.Fatalf("unexpected diff:\n%s", result.Unified)
	}
	if !hasSection(result.Sections, "Summary", "changed") || !hasSection(result.Sections, "Added", "added") {
		t.Fatalf("unexpected section changes: %+v", result.Sections)
	}
}

func TestComparePathsUsesDirectoryIndex(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "left/index.md", "# Report\nold\n")
	writeFile(t, root, "right/index.md", "# Report\nnew\n")
	result, err := ComparePaths(filepath.Join(root, "left"), filepath.Join(root, "right"), "left", "right")
	if err != nil {
		t.Fatalf("compare paths: %v", err)
	}
	if !strings.Contains(result.Unified, "+new") {
		t.Fatalf("unexpected diff:\n%s", result.Unified)
	}
}

func hasSection(changes []SectionChange, heading, status string) bool {
	for _, change := range changes {
		if change.Heading == heading && change.Status == status {
			return true
		}
	}
	return false
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
