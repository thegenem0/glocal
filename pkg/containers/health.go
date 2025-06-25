package containers

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

type ContainerHealthChecker struct {
	containers map[string]*ContainerInfo
	mu         sync.RWMutex
	logger     *zap.Logger
	client     *http.Client
}

func NewContainerHealthChecker(logger *zap.Logger) *ContainerHealthChecker {
	return &ContainerHealthChecker{
		containers: make(map[string]*ContainerInfo),
		logger:     logger,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (hc *ContainerHealthChecker) RegisterContainer(name string, containerInfo *ContainerInfo) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.containers[name] = containerInfo
}

func (hc *ContainerHealthChecker) DeregisterContainer(name string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	delete(hc.containers, name)
}

func (hc *ContainerHealthChecker) CheckHealth(ctx context.Context, name string) error {
	hc.mu.RLock()
	containerInfo, exists := hc.containers[name]
	hc.mu.RUnlock()

	if !exists {
		return fmt.Errorf("container %s not registered for health checking", name)
	}

	if containerInfo.Status != StatusRunning {
		return fmt.Errorf("container %s is not running (status: %s)", name, containerInfo.Status)
	}

	// If container had a health check endpoint
	if containerInfo.Config.WaitFor.Path != "" && containerInfo.Config.WaitFor.Port > 0 {
		return hc.checkHTTPHealth(ctx, containerInfo)
	}

	// Otherwise just check if container is up
	return hc.checkContainerRunning(containerInfo)
}

func (hc *ContainerHealthChecker) CheckAllHealth(ctx context.Context) map[string]error {
	hc.mu.RLock()
	containers := make(map[string]*ContainerInfo)
	for name, info := range hc.containers {
		containers[name] = info
	}
	hc.mu.RUnlock()

	results := make(map[string]error)
	for name := range containers {
		results[name] = hc.CheckHealth(ctx, name)
	}

	return results
}

func (hc *ContainerHealthChecker) checkHTTPHealth(ctx context.Context, containerInfo *ContainerInfo) error {
	port := containerInfo.Config.WaitFor.Port
	externalPort, exists := containerInfo.Ports[port]
	if !exists {
		return fmt.Errorf("health check port %d not found", port)
	}

	url := fmt.Sprintf("http://%s:%d%s", containerInfo.Host, externalPort, containerInfo.Config.WaitFor.Path)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := hc.client.Do(req)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}

func (hc *ContainerHealthChecker) checkContainerRunning(containerInfo *ContainerInfo) error {
	running := containerInfo.Container.IsRunning()

	if !running {
		return fmt.Errorf("container is not running")
	}

	return nil
}

