package config

// Config is the configuration for the sidecar
type Config struct {
	// Workspace is the workspace to sync
	Workspace string `json:"workspace"`
	// Sync is the configuration for the sync service
	Sync SyncConfig `json:"sync"`
	// Run is the configuration for the run service
	Run RunConfig `json:"run"`
}

// SyncConfig is the configuration for the sync service
type SyncConfig struct {
	// Type is the type of sync service to use
	Type string `json:"type"`
	// Git is the configuration for the git sync service
	Git GitSyncConfig `json:"git"`
}

// GitSyncConfig is the configuration for the git sync service
type GitSyncConfig struct {
	// Url is the repository to sync
	Url string `json:"url"`
	// Branch is the branch to sync
	Branch string `json:"branch"`
	// SyncInterval is the interval to sync
	SyncInterval string `json:"sync_interval"`
}

// RunConfig is the configuration for the run service
type RunConfig struct {
	// InitCmd is the command to init the workspace
	Init InitCmd `json:"init"`
	// RefreshCmd is the command to refresh the workspace
	Refresh []RefreshCmd `json:"refresh"`
}

// InitCmd is the configuration for the init command
type InitCmd struct {
	// Cmd is the command to init the workspace
	Cmds []string `json:"cmds"`
}

// RefreshCmd is the configuration for the refresh command
type RefreshCmd struct {
	// Condition is the condition to refresh the workspace
	Condition string `json:"condition"`
	// Cmd is the command to refresh the workspace
	Cmds []string `json:"cmds"`
}