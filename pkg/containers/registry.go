package containers

import (
	"sync"

	"github.com/thegenem0/glocal/pkg/config"
)

type ContainerRegistry struct {
	definitions map[string]config.ContainerConfig
	mu          sync.RWMutex
}

func NewContainerRegistry() *ContainerRegistry {
	return &ContainerRegistry{
		definitions: make(map[string]config.ContainerConfig),
	}
}

func (cr *ContainerRegistry) Register(name string, cfg config.ContainerConfig) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	cr.definitions[name] = cfg
}

func (cr *ContainerRegistry) Get(name string) (config.ContainerConfig, bool) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	cfg, exists := cr.definitions[name]
	return cfg, exists
}

func (cr *ContainerRegistry) List() map[string]config.ContainerConfig {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	result := make(map[string]config.ContainerConfig)
	for name, cfg := range cr.definitions {
		result[name] = cfg
	}

	return result
}

func (cr *ContainerRegistry) LoadFromConfig(containers map[string]config.ContainerConfig) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	for name, cfg := range containers {
		cr.definitions[name] = cfg
	}
}
