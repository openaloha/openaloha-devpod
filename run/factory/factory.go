package factory

import (
	"fmt"
	"sync"

	"openaloha.io/openaloha-devpod/run/handler"
)

// HandlerFactory 定义创建 handler 的工厂函数
type HandlerFactory func(workspace string) handler.RunHandler

var (
	providerMux sync.RWMutex
	providers   = make(map[string]HandlerFactory)
)

func Register(name string, factory HandlerFactory) {
	providerMux.Lock()
	defer providerMux.Unlock()
	providers[name] = factory
}

func New(name string, workspace string) (handler.RunHandler, error) {
	providerMux.RLock()
	defer providerMux.RUnlock()
	factory, ok := providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %s not found", name)
	}
	return factory(workspace), nil
}
