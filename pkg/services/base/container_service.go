package base

import (
	"context"
	"fmt"

	"github.com/thegenem0/glocal/pkg/config"
	"github.com/thegenem0/glocal/pkg/containers"
	"go.uber.org/zap"
)

type ContainerService struct {
	*BaseService
	containerName   string
	containerConfig config.ContainerConfig
	containerInfo   *containers.ContainerInfo
}

func NewContainerServcie(
	name string,
	containerName string,
	containerConfig config.ContainerConfig,
	containerMgr *containers.ContainerManager,
	logger *zap.Logger,
) *ContainerService {
	return &ContainerService{
		BaseService:     NewBaseService(name, containerMgr, logger),
		containerName:   containerName,
		containerConfig: containerConfig,
	}
}

func (cs *ContainerService) Initialize(ctx context.Context) error {
	cs.logger.Info("Initializing container service",
		zap.String("service", cs.name),
		zap.String("container", cs.containerName))

	containerInfo, err := cs.containerMgr.StartContainer(ctx, cs.containerName, cs.containerConfig)
	if err != nil {
		return fmt.Errorf("failed to start container %s: %w", cs.containerName, err)
	}

	cs.containerInfo = containerInfo

	cs.logger.Info("Container service initialized successfully",
		zap.String("service", cs.name),
		zap.String("container", cs.containerName),
		zap.String("host", containerInfo.Host),
		zap.Any("ports", containerInfo.Ports))

	return nil
}

func (cs *ContainerService) Stop(ctx context.Context) error {
	cs.logger.Info("Stopping container service",
		zap.String("service", cs.name),
		zap.String("container", cs.containerName))

	if err := cs.containerMgr.StopContainer(ctx, cs.containerName); err != nil {
		return fmt.Errorf("failed to stop container %s: %w", cs.containerName, err)
	}

	return nil
}

func (cs *ContainerService) Health(ctx context.Context) error {
	if cs.containerInfo == nil {
		return fmt.Errorf("container not initialized")
	}

	if cs.containerInfo.Status != containers.StatusRunning {
		return fmt.Errorf("container is not running: %s", cs.containerInfo.Status)
	}

	return nil
}

func (cs *ContainerService) GetContainerEndpoint(internalPort int) (string, error) {
	if cs.containerInfo == nil {
		return "", fmt.Errorf("container not initialized")
	}

	externalPort, exists := cs.containerInfo.Ports[internalPort]
	if !exists {
		return "", fmt.Errorf("port %d not found", internalPort)
	}

	return fmt.Sprintf("http://%s:%d", cs.containerInfo.Host, externalPort), nil
}

func (cs *ContainerService) GetContainer() *containers.ContainerInfo {
	return cs.containerInfo
}
