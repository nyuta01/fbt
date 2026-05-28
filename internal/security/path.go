package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func CleanProjectRelative(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("absolute paths are not allowed")
	}
	clean := filepath.Clean(path)
	if clean == "." {
		return "", fmt.Errorf("path is empty")
	}
	for _, part := range strings.Split(clean, string(filepath.Separator)) {
		if part == ".." {
			return "", fmt.Errorf("path must not contain .. segments")
		}
	}
	return clean, nil
}

func ResolveProjectRelative(root, path string) (string, error) {
	clean, err := CleanProjectRelative(path)
	if err != nil {
		return "", err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	return filepath.Join(rootAbs, clean), nil
}

func IsWithin(parent, child string) bool {
	parentAbs, err := filepath.Abs(parent)
	if err != nil {
		return false
	}
	childAbs, err := filepath.Abs(child)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(parentAbs, childAbs)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func RequireWithin(parent, child string) error {
	if !IsWithin(parent, child) {
		return fmt.Errorf("%s is outside %s", child, parent)
	}
	return nil
}

func RejectSymlinkPath(root, target string) error {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	if err := RequireWithin(rootAbs, targetAbs); err != nil {
		return err
	}

	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return err
	}
	if rel == "." {
		return rejectIfSymlink(rootAbs)
	}

	current := rootAbs
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		if err := rejectIfSymlink(current); err != nil {
			return err
		}
	}
	return nil
}

func rejectIfSymlink(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("symlink rejected: %s", path)
	}
	return nil
}
