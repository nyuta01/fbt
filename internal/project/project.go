package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const ConfigFileName = "fs_project.yml"

type Project struct {
	RootDir    string
	ConfigPath string
}

func Open(projectDir string) (Project, error) {
	if projectDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return Project{}, err
		}
		projectDir = wd
	}

	abs, err := filepath.Abs(projectDir)
	if err != nil {
		return Project{}, err
	}

	info, err := os.Stat(abs)
	if err != nil {
		return Project{}, err
	}
	if !info.IsDir() {
		if filepath.Base(abs) == ConfigFileName {
			return Project{RootDir: filepath.Dir(abs), ConfigPath: abs}, nil
		}
		return Project{}, fmt.Errorf("project path is not a directory or %s: %s", ConfigFileName, abs)
	}

	configPath := filepath.Join(abs, ConfigFileName)
	if _, err := os.Stat(configPath); err == nil {
		return Project{RootDir: abs, ConfigPath: configPath}, nil
	}

	return Discover(abs)
}

func Discover(start string) (Project, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return Project{}, err
	}

	info, err := os.Stat(abs)
	if err != nil {
		return Project{}, err
	}
	if !info.IsDir() {
		abs = filepath.Dir(abs)
	}

	for {
		configPath := filepath.Join(abs, ConfigFileName)
		if _, err := os.Stat(configPath); err == nil {
			return Project{RootDir: abs, ConfigPath: configPath}, nil
		}

		parent := filepath.Dir(abs)
		if parent == abs {
			break
		}
		abs = parent
	}

	return Project{}, errors.New("could not find fs_project.yml")
}
