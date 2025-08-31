package factory

import (
	"fmt"
	"sync"

	"openaloha.io/openaloha-devpod/sync/handler"
)

var (
	providerMu sync.RWMutex
	providers  = make(map[string]handler.SyncHandler)
)

func Register(name string, provider handler.SyncHandler) {
	providerMu.Lock()
	defer providerMu.Unlock()
	providers[name] = provider
}

func New(name string) (handler.SyncHandler, error) {
	providerMu.RLock()
	defer providerMu.RUnlock()
	provider, ok := providers[name]
	if !ok {
		return nil, fmt.Errorf("sync handler %s not found", name)
	}
	return provider, nil
}
