package plugin

import (
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

const ManifestFileName = "fbt_plugin.yml"

type Manifest struct {
	Name     string            `yaml:"name" json:"name"`
	Version  string            `yaml:"version" json:"version"`
	Protocol string            `yaml:"protocol" json:"protocol"`
	Command  string            `yaml:"command" json:"command"`
	Provides []ProvidedRunner  `yaml:"provides" json:"provides"`
	Env      []string          `yaml:"env" json:"env,omitempty"`
	Checksum map[string]string `yaml:"checksum" json:"checksum,omitempty"`
	Path     string            `yaml:"-" json:"path"`
	RootDir  string            `yaml:"-" json:"root_dir"`
}

type ProvidedRunner struct {
	Runner         string   `yaml:"runner" json:"runner"`
	Type           string   `yaml:"type" json:"type"`
	TransformTypes []string `yaml:"transform_types" json:"transform_types,omitempty"`
	ArtifactTypes  []string `yaml:"artifact_types" json:"artifact_types,omitempty"`
}

func Load(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, err
	}
	manifest.Path = path
	manifest.RootDir = filepath.Dir(path)
	return manifest, nil
}

func LoadAll(root string) ([]Manifest, error) {
	pattern := filepath.Join(root, "*", ManifestFileName)
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	manifests := make([]Manifest, 0, len(paths))
	for _, path := range paths {
		manifest, err := Load(path)
		if err != nil {
			return nil, err
		}
		manifests = append(manifests, manifest)
	}
	return manifests, nil
}
