package config

import (
	"io/ioutil"
	"gopkg.in/yaml.v2"
)

// Config is the configuration for the sidecar
type Config struct {
	// Workspace is the workspace to sync
	Workspace string `yaml:"workspace"`
	// Sync is the configuration for the sync service
	Sync SyncConfig `yaml:"sync"`
	// Run is the configuration for the run service
	Run RunConfig `yaml:"run"`
	// Coding is the configuration for the coding service
	Coding CodingConfig `yaml:"coding"`
}

// LoadFromFile loads configuration from YAML file
func LoadFromFile(filePath string) (*Config, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// SyncConfig is the configuration for the sync service
type SyncConfig struct {
	// Type is the type of sync service to use
	Type string `yaml:"type"`
	// Git is the configuration for the git sync service
	Git GitSyncConfig `yaml:"git"`
}

// GitSyncConfig is the configuration for the git sync service
type GitSyncConfig struct {
	// Url is the repository to sync
	Url string `yaml:"url"`
	// Branch is the branch to sync
	Branch string `yaml:"branch"`
	// SyncInterval is the interval to sync
	SyncInterval string `yaml:"sync-interval"`
}

// RunConfig is the configuration for the run service
type RunConfig struct {
	// InitCmd is the command to init the workspace
	Init InitCmd `yaml:"init"`
	// RefreshCmd is the command to refresh the workspace
	Refresh []RefreshCmd `yaml:"refresh"`
}

// InitCmd is the configuration for the init command
type InitCmd struct {
	// Cmd is the command to init the workspace
	Cmds []string `yaml:"cmds"`
}

// RefreshCmd is the configuration for the refresh command
type RefreshCmd struct {
	// Condition is the condition to refresh the workspace
	Condition string `yaml:"condition"`
	// Cmd is the command to refresh the workspace
	Cmds []string `yaml:"cmds"`
}

type CodingConfig struct {
	Type string `yaml:"type"`
	Gemini GeminiCodingConfig `yaml:"gemini"`
}

type GeminiCodingConfig struct {
	Baseurl string `yaml:"baseurl"`
	Apikey string `yaml:"apikey"`
}