package runtime

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thegenem0/glocal/pkg/containers"
	"go.uber.org/zap"
)

type Server struct {
	router       *gin.Engine
	registry     *ServiceRegistry
	containerMgr *containers.ContainerManager
	logger       *zap.Logger
	port         int
}

func NewServer(port int, logger *zap.Logger) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	return &Server{
		router:       router,
		registry:     NewServiceRegistry(),
		containerMgr: containers.NewContainerManager(logger),
		logger:       logger,
		port:         port,
	}
}

func (s *Server) RegisterService(service Service) {
	s.registry.Register(service)

	for _, route := range service.Routes() {
		s.router.Any(route, gin.WrapH(service.Handler()))
		s.router.Any(route+"/*path", gin.WrapH(service.Handler()))

	}
}

func (s *Server) GetContainerManager() *containers.ContainerManager {
	return s.containerMgr
}

func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting GLocal server", zap.Int("port", s.port))

	for name, service := range s.registry.All() {
		s.logger.Info("Initializing service", zap.String("service", name))

		if err := service.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize service %s: %w", name, err)
		}
	}

	for name, service := range s.registry.All() {
		s.logger.Info("Starting service", zap.String("service", name))

		if err := service.Start(ctx); err != nil {
			return fmt.Errorf("failed to start service %s: %w", name, err)
		}
	}

	s.router.GET("/health", s.healthHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: s.router,
	}

	serverErr := make(chan error, 1)
	go func() {
		s.logger.Info("HTTP server listening", zap.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	select {
	case <-ctx.Done():
		s.logger.Info("Shutting down server due to context cancellation")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("Error during server shutdown", zap.Error(err))
		}

		for name, service := range s.registry.All() {
			s.logger.Info("Stopping service", zap.String("service", name))
			if err := service.Stop(shutdownCtx); err != nil {
				s.logger.Error("Error stopping service",
					zap.String("service", name),
					zap.Error(err))
			}
		}

		if err := s.containerMgr.StopAll(shutdownCtx); err != nil {
			s.logger.Error("Error stopping containers", zap.Error(err))
		}

		return ctx.Err()

	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	}
}

func (s *Server) healthHandler(c *gin.Context) {
	ctx := c.Request.Context()
	status := make(map[string]string)

	for name, service := range s.registry.All() {
		if err := service.Health(ctx); err != nil {
			status[name] = "degraded: " + err.Error()
		} else {
			status[name] = "healthy"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "ok",
		"services": status,
	})
}
