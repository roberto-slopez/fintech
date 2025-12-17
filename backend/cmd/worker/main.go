package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/fintech-multipass/backend/internal/infrastructure/config"
	"github.com/fintech-multipass/backend/internal/infrastructure/database"
	"github.com/fintech-multipass/backend/internal/infrastructure/logger"
	"github.com/fintech-multipass/backend/internal/infrastructure/queue"
)

// Worker principal para procesamiento asíncrono de trabajos
func main() {
	log := logger.NewLogger()
	log.Info().Msg("Starting Fintech Worker...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Initialize database connection
	db, err := database.NewPostgresConnection(cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// Initialize job queue
	jobQueue := queue.NewPostgresQueue(db, log)

	// Start workers
	ctx, cancel := context.WithCancel(context.Background())
	jobQueue.StartWorkers(ctx, cfg.Queue.WorkerCount)

	log.Info().Int("workers", cfg.Queue.WorkerCount).Msg("Workers started")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down workers...")
	cancel()
	log.Info().Msg("Workers stopped")
}

