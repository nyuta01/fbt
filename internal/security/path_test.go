package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCleanProjectRelativeRejectsEscapes(t *testing.T) {
	for _, path := range []string{"../x", "a/../../b", "/tmp/x"} {
		if _, err := CleanProjectRelative(path); err == nil {
			t.Fatalf("expected %q to be rejected", path)
		}
	}
	got, err := CleanProjectRelative("a/./b")
	if err != nil {
		t.Fatalf("clean path: %v", err)
	}
	if got != filepath.Join("a", "b") {
		t.Fatalf("unexpected clean path: %q", got)
	}
}

func TestRequireWithin(t *testing.T) {
	root := t.TempDir()
	if err := RequireWithin(root, filepath.Join(root, "child")); err != nil {
		t.Fatalf("expected child to be within root: %v", err)
	}
	if err := RequireWithin(root, filepath.Dir(root)); err == nil {
		t.Fatal("expected parent directory to be rejected")
	}
}

func TestRejectSymlinkPath(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.txt")
	if err := os.WriteFile(target, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := RejectSymlinkPath(root, link); err == nil {
		t.Fatal("expected symlink to be rejected")
	}
}
