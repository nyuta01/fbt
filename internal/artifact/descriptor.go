package artifact

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nyuta01/fbt/internal/config"
	"github.com/nyuta01/fbt/internal/security"
)

type Descriptor struct {
	MediaType    string `json:"media_type"`
	Digest       string `json:"digest"`
	Size         *int64 `json:"size"`
	ArtifactType string `json:"artifact_type"`
	FileCount    int    `json:"file_count,omitempty"`
}

type directoryEntry struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Size   int64  `json:"size"`
	Digest string `json:"digest"`
}

func Describe(root, path, artifactAlias string) (Descriptor, error) {
	artifactType, ok := config.LookupArtifactType(artifactAlias)
	if !ok {
		return Descriptor{}, fmt.Errorf("standard descriptor unsupported for artifact type %q", artifactAlias)
	}

	abs, err := security.ResolveProjectRelative(root, path)
	if err != nil {
		return Descriptor{}, err
	}
	if err := security.RejectSymlinkPath(root, abs); err != nil {
		return Descriptor{}, err
	}

	info, err := os.Lstat(abs)
	if err != nil {
		return Descriptor{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return Descriptor{}, fmt.Errorf("symlink rejected: %s", path)
	}

	switch artifactType.PathKind {
	case config.PathKindFile:
		if info.IsDir() {
			return Descriptor{}, fmt.Errorf("artifact type %q expects a file, got directory", artifactAlias)
		}
		return describeFile(abs, artifactType.Descriptor)
	case config.PathKindDirectory:
		if !info.IsDir() {
			return Descriptor{}, fmt.Errorf("artifact type %q expects a directory, got file", artifactAlias)
		}
		return describeDirectory(abs, artifactType.Descriptor)
	default:
		return Descriptor{}, fmt.Errorf("artifact type %q has unsupported path kind %q", artifactAlias, artifactType.PathKind)
	}
}

func describeFile(path, descriptorType string) (Descriptor, error) {
	digest, size, err := fileDigest(path)
	if err != nil {
		return Descriptor{}, err
	}
	return Descriptor{
		MediaType:    mediaTypeForDescriptor(descriptorType),
		Digest:       digest,
		Size:         &size,
		ArtifactType: descriptorType,
	}, nil
}

func describeDirectory(root, descriptorType string) (Descriptor, error) {
	entries := []directoryEntry{}
	var totalSize int64
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink rejected: %s", path)
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported non-regular file in descriptor: %s", path)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		normalized := filepath.ToSlash(rel)
		if filepath.IsAbs(normalized) || strings.Contains(normalized, "../") || normalized == ".." {
			return fmt.Errorf("invalid directory entry path: %s", normalized)
		}
		digest, size, err := fileDigest(path)
		if err != nil {
			return err
		}
		entries = append(entries, directoryEntry{
			Path:   normalized,
			Kind:   "file",
			Size:   size,
			Digest: digest,
		})
		totalSize += size
		return nil
	})
	if err != nil {
		return Descriptor{}, err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})
	data, err := json.Marshal(entries)
	if err != nil {
		return Descriptor{}, err
	}
	sum := sha256.Sum256(data)
	return Descriptor{
		MediaType:    "inode/directory",
		Digest:       "sha256:" + hex.EncodeToString(sum[:]),
		Size:         &totalSize,
		ArtifactType: descriptorType,
		FileCount:    len(entries),
	}, nil
}

func fileDigest(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return "", 0, err
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), size, nil
}

func VersionID(projectName, artifactName, digest string) (string, error) {
	const prefix = "sha256:"
	if !strings.HasPrefix(digest, prefix) {
		return "", fmt.Errorf("digest must start with %s", prefix)
	}
	hexDigest := strings.TrimPrefix(digest, prefix)
	if len(hexDigest) != 64 {
		return "", fmt.Errorf("sha256 digest must contain 64 hex characters")
	}
	for _, r := range hexDigest {
		if !strings.ContainsRune("0123456789abcdef", r) {
			return "", fmt.Errorf("sha256 digest must be lowercase hex")
		}
	}
	return fmt.Sprintf("artifact_version.%s.%s.sha256_%s", projectName, artifactName, hexDigest), nil
}

func mediaTypeForDescriptor(descriptorType string) string {
	switch descriptorType {
	case "fbt.artifact.text_file.v1":
		return "text/plain; charset=utf-8"
	case "fbt.artifact.markdown_document.v1":
		return "text/markdown; charset=utf-8"
	case "fbt.artifact.docx_document.v1":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case "fbt.artifact.xlsx_workbook.v1":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case "fbt.artifact.pdf_document.v1":
		return "application/pdf"
	case "fbt.artifact.html_document.v1":
		return "text/html; charset=utf-8"
	case "fbt.artifact.json_document.v1":
		return "application/json"
	case "fbt.artifact.binary_file.v1":
		return "application/octet-stream"
	default:
		return "application/octet-stream"
	}
}
