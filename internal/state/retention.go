package state

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
)

type RetentionReport struct {
	Policy              string   `json:"policy"`
	StateDir            string   `json:"state_dir"`
	ArtifactDir         string   `json:"artifact_dir"`
	ArchiveUnit         string   `json:"archive_unit"`
	StateBytes          int64    `json:"state_bytes"`
	ArtifactBytes       int64    `json:"artifact_bytes"`
	RunRecords          int      `json:"run_records"`
	ArtifactVersions    int      `json:"artifact_versions"`
	CurrentVersions     int      `json:"current_versions"`
	HistoricalVersions  int      `json:"historical_versions"`
	CurrentVersionIDs   []string `json:"current_version_ids,omitempty"`
	ProtectedVersionIDs []string `json:"protected_version_ids,omitempty"`
	MissingStorage      []string `json:"missing_storage,omitempty"`
	ArchiveRoots        []string `json:"archive_roots"`
	PruneSupported      bool     `json:"prune_supported"`
	DryRunRequired      bool     `json:"dry_run_required"`
}

func BuildRetentionReport(projectDir string, store Store) (RetentionReport, error) {
	snapshot, err := store.ReadState()
	if err != nil {
		return RetentionReport{}, err
	}
	versions, err := store.ReadArtifactVersions()
	if err != nil {
		return RetentionReport{}, err
	}
	runRecords, err := store.ReadRunResults()
	if err != nil {
		return RetentionReport{}, err
	}

	artifactDir := filepath.Join(projectDir, ".fbt", "artifacts")
	stateBytes, err := directoryBytes(store.Dir)
	if err != nil {
		return RetentionReport{}, err
	}
	artifactBytes, err := directoryBytes(artifactDir)
	if err != nil {
		return RetentionReport{}, err
	}

	current := map[string]bool{}
	for _, pointer := range snapshot.CurrentArtifacts {
		if pointer.CurrentVersionID != "" {
			current[pointer.CurrentVersionID] = true
		}
	}
	currentIDs := make([]string, 0, len(current))
	for versionID := range current {
		currentIDs = append(currentIDs, versionID)
	}
	sort.Strings(currentIDs)
	currentInIndex := 0
	missingStorage := []string{}
	for versionID, version := range versions.ArtifactVersions {
		if current[versionID] {
			currentInIndex++
		}
		if version.StoragePath == "" {
			missingStorage = append(missingStorage, versionID)
			continue
		}
		if _, err := os.Stat(filepath.Join(projectDir, filepath.FromSlash(version.StoragePath))); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				missingStorage = append(missingStorage, versionID)
				continue
			}
			return RetentionReport{}, err
		}
	}
	sort.Strings(missingStorage)

	historical := len(versions.ArtifactVersions) - currentInIndex
	if historical < 0 {
		historical = 0
	}

	return RetentionReport{
		Policy:              "keep_all",
		StateDir:            store.Dir,
		ArtifactDir:         artifactDir,
		ArchiveUnit:         "state_and_artifacts",
		StateBytes:          stateBytes,
		ArtifactBytes:       artifactBytes,
		RunRecords:          len(runRecords),
		ArtifactVersions:    len(versions.ArtifactVersions),
		CurrentVersions:     currentInIndex,
		HistoricalVersions:  historical,
		CurrentVersionIDs:   currentIDs,
		ProtectedVersionIDs: currentIDs,
		MissingStorage:      missingStorage,
		ArchiveRoots:        []string{store.Dir, artifactDir},
		PruneSupported:      false,
		DryRunRequired:      true,
	}, nil
}

func directoryBytes(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrNotExist) && path == root {
				return nil
			}
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			total += info.Size()
		}
		return nil
	})
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	return total, err
}
