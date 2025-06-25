package services

import (
	"context"
	"net/http"

	"github.com/thegenem0/glocal/pkg/containers"
	"go.uber.org/zap"
)

type BaseService struct {
	name         string
	containerMgr *containers.ContainerManager
	logger       *zap.Logger
	routes       []string
}

func NewBaseService(name string, containerMgr *containers.ContainerManager, logger *zap.Logger) *BaseService {
	return &BaseService{
		name:         name,
		containerMgr: containerMgr,
		logger:       logger,
		routes:       []string{},
	}
}

func (bs *BaseService) Name() string {
	return bs.name
}

func (bs *BaseService) Routes() []string {
	return bs.routes
}

func (bs *BaseService) SetRoutes(routes []string) {
	bs.routes = routes
}

func (bs *BaseService) Initialize(ctx context.Context) error {
	bs.logger.Info("Initializing base service", zap.String("service", bs.name))
	return nil
}

func (bs *BaseService) Start(ctx context.Context) error {
	bs.logger.Info("Starting base service", zap.String("service", bs.name))
	return nil
}

func (bs *BaseService) Stop(ctx context.Context) error {
	bs.logger.Info("Stopping base service", zap.String("service", bs.name))
	return nil
}

func (bs *BaseService) Health(ctx context.Context) error {
	return nil
}

func (bs *BaseService) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		_, _ = w.Write([]byte("Service not implemented"))
	})
}
