package factory

import (
	"fmt"
	"sync"

	"openaloha.io/openaloha-devpod/coding/handler"
)

var (
	providerMu sync.RWMutex
	providers  = make(map[string]handler.CodingHandler)
)

func Register(name string, provider handler.CodingHandler) {
	providerMu.Lock()
	defer providerMu.Unlock()
	providers[name] = provider
}

func New(name string) (handler.CodingHandler, error) {
	providerMu.RLock()
	defer providerMu.RUnlock()
	provider, ok := providers[name]
	if !ok {
		return nil, fmt.Errorf("coding handler %s not found", name)
	}
	return provider, nil
}
