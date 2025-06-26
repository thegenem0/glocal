package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/thegenem0/glocal/pkg/config"
	"github.com/thegenem0/glocal/pkg/runtime"
	"github.com/thegenem0/glocal/pkg/services/storage"
	"go.uber.org/zap"
)

func main() {
	var configPath = flag.String("config", "configs/default.yaml", "Path to configuration file")
	flag.Parse()

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Sync()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	server := runtime.NewServer(cfg.Server.Port, logger)

	// TODO(thegenem0):
	// Register services based on config

	if err := registerServices(server, cfg, logger); err != nil {
		logger.Fatal("Failed to register services", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Shutdown signal received")
		cancel()
	}()

	logger.Info("Starting GLocal server")
	if err := server.Start(ctx); err != nil {
		if err == context.Canceled {
			logger.Info("Server shutdown completed")
		} else {
			logger.Fatal("Server failed to start", zap.Error(err))
		}
	}

	logger.Info("GLocal server stopped")
}

func registerServices(server *runtime.Server, cfg *config.Config, logger *zap.Logger) error {
	containerMgr := server.GetContainerManager()

	containerMgr.GetRegistry().LoadFromConfig(cfg.Containers)

	if serviceConfig, exists := cfg.Services["storage"]; exists && serviceConfig.Enabled {
		containerConfig, exists := cfg.Containers[serviceConfig.Container]
		if !exists {
			logger.Error("Container configuration not found",
				zap.String("container", serviceConfig.Container))
			return nil // Continue with other services
		}

		storageService := storage.NewStorageService(containerMgr, containerConfig, logger)
		server.RegisterService(storageService)
		logger.Info("Storage service registered")
	}

	return nil
}
