package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ialekseychuk/5_services/internal/config"
	"github.com/ialekseychuk/5_services/internal/logger"
	"github.com/ialekseychuk/5_services/internal/service"
)

func main() {
	logger := logger.NewLogger()

	config, err := config.LoadConfig()
	if err != nil {
		logger.Fatalf("Failed to load config: %v", err)
	}

	svc := service.NewService(config, logger)

	go func() {
		logger.Infof("Starting service %s on port %d", config.ID, config.Port)
		if err := svc.Run(); err != nil {
			logger.Errorf("Failed to start service: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down service...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	svc.Shutdown(ctx)

	logger.Info("Service shutdown complete")
}
