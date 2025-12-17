package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fintech-multipass/backend/internal/infrastructure/config"
	"github.com/fintech-multipass/backend/internal/infrastructure/database"
	"github.com/fintech-multipass/backend/internal/infrastructure/logger"
	"github.com/fintech-multipass/backend/internal/interfaces/http/router"
	"github.com/fintech-multipass/backend/internal/infrastructure/cache"
	"github.com/fintech-multipass/backend/internal/infrastructure/queue"
	"github.com/joho/godotenv"
)

// @title Fintech Multipaís API
// @version 1.0
// @description API para gestión de solicitudes de crédito multipaís
// @host localhost:8080
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	// Load .env file if it exists (for local development)
	if err := godotenv.Load("../.env"); err != nil {
		// .env file not found or error loading it, continue with system environment variables
		if err := godotenv.Load(); err != nil {
			// También intentar desde el directorio actual
		}
	}

	// Initialize logger
	log := logger.NewLogger()
	log.Info().Msg("Starting Fintech Multipaís API...")

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
	log.Info().Msg("Database connection established")

	// Initialize cache
	var cacheClient cache.CacheService
	redisCache, err := cache.NewRedisCache(cfg.Cache)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to connect to Redis, using in-memory cache")
		cacheClient = cache.NewMemoryCache()
	} else {
		cacheClient = redisCache
	}
	defer cacheClient.Close()

	// Initialize job queue
	jobQueue := queue.NewPostgresQueue(db, log)
	
	// Start queue workers
	workerCtx, workerCancel := context.WithCancel(context.Background())
	jobQueue.StartWorkers(workerCtx, cfg.Queue.WorkerCount)

	// Setup router with all dependencies
	r := router.NewRouter(db, cacheClient, jobQueue, cfg, log)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Info().Int("port", cfg.Server.Port).Msg("Server starting...")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed to start")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down server...")

	// Cancel worker context
	workerCancel()

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited properly")
}

