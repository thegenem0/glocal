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
