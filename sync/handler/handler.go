package handler

import (
	"github.com/openaloha/openaloha-devpod/config"
	"github.com/openaloha/openaloha-devpod/runfunc"
)

// InitFunc is the function type for initialization
type InitFunc func() error

// RefreshFunc is the function type for refresh
type RefreshFunc func() error

// SyncHandler is the interface for the sync handler
type SyncHandler interface {
	// Init is the method to initialize code
	Init(workspace string, syncConfig config.SyncConfig, initFunc runfunc.InitFunc) error

	// Refresh is the method to refresh code
	Refresh(workspace string, syncConfig config.SyncConfig, refreshFunc runfunc.RefreshFunc) error
}
