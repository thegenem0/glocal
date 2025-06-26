package containers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"maps"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/thegenem0/glocal/pkg/config"
	"go.uber.org/zap"
)

type ContainerStatus string

const (
	StatusStarting ContainerStatus = "starting"
	StatusRunning  ContainerStatus = "running"
	StatusStopping ContainerStatus = "stopping"
	StatusStopped  ContainerStatus = "stopped"
	StatusError    ContainerStatus = "error"
)

type ContainerManager struct {
	containers map[string]*ContainerInfo
	registry   *ContainerRegistry
	health     *ContainerHealthChecker
	mu         sync.RWMutex
	logger     *zap.Logger
}

type ContainerInfo struct {
	Container testcontainers.Container
	Host      string
	Ports     map[int]int // internal -> external mapping
	Name      string
	Image     string
	Status    ContainerStatus
	StartedAt time.Time
	Config    config.ContainerConfig
}

func NewContainerManager(logger *zap.Logger) *ContainerManager {
	return &ContainerManager{
		containers: make(map[string]*ContainerInfo),
		registry:   NewContainerRegistry(),
		health:     NewContainerHealthChecker(logger),
		logger:     logger,
	}
}

func (cm *ContainerManager) StartContainer(ctx context.Context, name string, cfg config.ContainerConfig) (*ContainerInfo, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.containers[name]; exists {
		return nil, fmt.Errorf("container %s already exists", name)
	}

	cm.logger.Info("Starting container",
		zap.String("name", name),
		zap.String("image", cfg.Image))

	req, err := cm.buildContainerRequest(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build container request: %w", err)
	}

	containerInfo := &ContainerInfo{
		Name:      name,
		Image:     cfg.Image,
		Status:    StatusStarting,
		Config:    cfg,
		Ports:     make(map[int]int),
		StartedAt: time.Now(),
	}

	cm.containers[name] = containerInfo

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: *req,
		Started:          true,
	})

	if err != nil {
		containerInfo.Status = StatusError
		return nil, fmt.Errorf("failed to start container %s: %w", name, err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		containerInfo.Status = StatusError
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	for _, internalPort := range cfg.Ports {
		configPort, err := nat.NewPort("tcp", fmt.Sprint(internalPort))
		if err != nil {
			containerInfo.Status = StatusError
			return nil, fmt.Errorf("failed to parse %d as nat.Port: %w", internalPort, err)
		}

		mappedPort, err := container.MappedPort(ctx, configPort)
		if err != nil {
			containerInfo.Status = StatusError
			return nil, fmt.Errorf("failed to get mapped port %d: %w", internalPort, err)
		}

		containerInfo.Ports[internalPort] = mappedPort.Int()
	}

	containerInfo.Container = container
	containerInfo.Host = host
	containerInfo.Status = StatusRunning

	cm.health.RegisterContainer(name, containerInfo)

	cm.logger.Info("Container started successfully",
		zap.String("name", name),
		zap.String("host", host),
		zap.Any("ports", containerInfo.Ports))

	return containerInfo, nil
}

func (cm *ContainerManager) GetContainer(name string) (*ContainerInfo, bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	container, exists := cm.containers[name]
	return container, exists
}

func (cm *ContainerManager) GetRegistry() *ContainerRegistry {
	return cm.registry
}

func (cm *ContainerManager) GetContainerEndpoint(name string, internalPort int) (string, error) {
	containerInfo, exists := cm.GetContainer(name)
	if !exists {
		return "", fmt.Errorf("container %s not found", name)
	}

	externalPort, exists := containerInfo.Ports[internalPort]
	if !exists {
		return "", fmt.Errorf("port %d not found for container %s", internalPort, name)

	}

	return fmt.Sprintf("http://%s:%d", containerInfo.Host, externalPort), nil
}

func (cm *ContainerManager) ListContainers() map[string]*ContainerInfo {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	result := make(map[string]*ContainerInfo)
	maps.Copy(result, cm.containers)

	return result
}

func (cm *ContainerManager) StopContainer(ctx context.Context, name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	containerInfo, exists := cm.containers[name]
	if !exists {
		return fmt.Errorf("container %s not found", name)
	}

	cm.logger.Info("Stopping container", zap.String("name", name))
	containerInfo.Status = StatusStopping

	cm.health.DeregisterContainer(name)

	if err := containerInfo.Container.Terminate(ctx); err != nil {
		containerInfo.Status = StatusError
		return fmt.Errorf("failed to stop container %s: %w", name, err)
	}

	containerInfo.Status = StatusStopped
	delete(cm.containers, name)

	cm.logger.Info("Container stopped successfully", zap.String("name", name))
	return nil
}

func (cm *ContainerManager) StopAll(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for name := range cm.containers {
		if err := cm.stopContainerUnsafe(ctx, name); err != nil {
			cm.logger.Error("Failed to stop container",
				zap.String("name", name),
				zap.Error(err))
		}
	}

	return nil
}

func (cm *ContainerManager) buildContainerRequest(cfg config.ContainerConfig) (*testcontainers.ContainerRequest, error) {
	exposedPorts := make([]string, len(cfg.Ports))
	for i, port := range cfg.Ports {
		exposedPorts[i] = fmt.Sprintf("%d/tcp", port)
	}

	req := &testcontainers.ContainerRequest{
		Image:        cfg.Image,
		ExposedPorts: exposedPorts,
		Env:          cfg.Environment,
		Cmd:          cfg.Cmd,
	}

	if cfg.WaitFor.Port > 0 {
		port, err := nat.NewPort("tcp", fmt.Sprint(cfg.WaitFor.Port))
		if cfg.WaitFor.Path != "" {
			if err != nil {
				return nil, err

			}
			req.WaitingFor = wait.ForHTTP(cfg.WaitFor.Path).WithPort(port)
		} else {
			req.WaitingFor = wait.ForListeningPort(port)
		}
	}

	return req, nil
}

func (cm *ContainerManager) stopContainerUnsafe(ctx context.Context, name string) error {
	containerInfo := cm.containers[name]
	containerInfo.Status = StatusStopping

	cm.health.DeregisterContainer(name)

	if err := containerInfo.Container.Terminate(ctx); err != nil {
		containerInfo.Status = StatusError
		return err
	}

	delete(cm.containers, name)
	return nil
}
