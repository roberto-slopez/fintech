package router

import (
	"net/http"

	"github.com/fintech-multipass/backend/internal/application/usecase"
	"github.com/fintech-multipass/backend/internal/domain/entity"
	"github.com/fintech-multipass/backend/internal/infrastructure/cache"
	"github.com/fintech-multipass/backend/internal/infrastructure/config"
	"github.com/fintech-multipass/backend/internal/infrastructure/database"
	"github.com/fintech-multipass/backend/internal/infrastructure/logger"
	"github.com/fintech-multipass/backend/internal/infrastructure/persistence"
	"github.com/fintech-multipass/backend/internal/infrastructure/queue"
	"github.com/fintech-multipass/backend/internal/interfaces/http/handler"
	"github.com/fintech-multipass/backend/internal/interfaces/http/middleware"
	"github.com/fintech-multipass/backend/internal/interfaces/websocket"
	"github.com/gin-gonic/gin"
)

// NewRouter crea y configura el router principal
func NewRouter(
	db *database.PostgresDB,
	cacheService cache.CacheService,
	jobQueue *queue.PostgresQueue,
	cfg *config.Config,
	log *logger.Logger,
) *gin.Engine {
	// Configurar modo de Gin
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Middlewares globales
	r.Use(middleware.Recovery(log))
	r.Use(middleware.RequestLogger(log))
	r.Use(middleware.CORSConfig(cfg.Server.CorsOrigins))
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.ErrorHandler(log))

	// Inicializar repositorios
	countryRepo := persistence.NewCountryRepository(db)
	appRepo := persistence.NewApplicationRepository(db)
	userRepo := persistence.NewUserRepository(db)

	// Inicializar casos de uso
	authUseCase := usecase.NewAuthUseCase(userRepo, cfg.JWT, log)
	countryUseCase := usecase.NewCountryUseCase(countryRepo, cacheService, log)
	appUseCase := usecase.NewApplicationUseCase(
		appRepo,
		countryRepo,
		nil, // providerRepo - se puede agregar después
		nil, // validator - se puede agregar después
		cacheService,
		nil, // eventPub - se puede agregar después
		nil, // jobQueue - se puede agregar después
		log,
	)

	// Inicializar handlers
	authHandler := handler.NewAuthHandler(authUseCase, log)
	countryHandler := handler.NewCountryHandler(countryUseCase, log)
	appHandler := handler.NewApplicationHandler(appUseCase, log)
	webhookHandler := handler.NewWebhookHandler(db, log, cfg.Webhook)

	// Inicializar middleware de autenticación
	authMiddleware := middleware.NewAuthMiddleware(authUseCase)

	// Inicializar WebSocket hub
	wsHub := websocket.NewHub(log)
	go wsHub.Run()

	// ==========================================
	// RUTAS PÚBLICAS (sin autenticación)
	// ==========================================

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "fintech-multipass-api",
		})
	})

	// Ready check (con verificación de dependencias)
	r.GET("/ready", func(c *gin.Context) {
		if err := db.HealthCheck(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not_ready",
				"error":  "database connection failed",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// ==========================================
	// API v1
	// ==========================================
	v1 := r.Group("/api/v1")

	// Auth routes (públicas)
	auth := v1.Group("/auth")
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/register", authHandler.Register)
		auth.POST("/refresh", authHandler.RefreshToken)
	}

	// Countries routes (públicas, solo lectura)
	countries := v1.Group("/countries")
	{
		countries.GET("", countryHandler.GetAll)
		countries.GET("/:code", countryHandler.GetByCode)
		countries.GET("/:code/document-types", countryHandler.GetDocumentTypes)
	}

	// Webhook endpoint (público, con verificación de firma)
	v1.POST("/webhooks/:source", webhookHandler.HandleIncoming)

	// ==========================================
	// RUTAS PROTEGIDAS (requieren autenticación)
	// ==========================================
	protected := v1.Group("")
	protected.Use(authMiddleware.Authenticate())

	// Auth routes protegidas
	protected.GET("/auth/me", authHandler.Me)

	// Countries rules (protegida)
	protected.GET("/countries/:code/rules", countryHandler.GetRules)

	// Applications routes
	applications := protected.Group("/applications")
	{
		// Crear solicitud (requiere permiso 'create')
		applications.POST("", authMiddleware.RequirePermission("create"), appHandler.Create)

		// Listar solicitudes (requiere permiso 'read')
		applications.GET("", authMiddleware.RequirePermission("read"), appHandler.List)

		// Obtener solicitud por ID
		applications.GET("/:id", authMiddleware.RequirePermission("read"), appHandler.GetByID)

		// Obtener historial de una solicitud
		applications.GET("/:id/history", authMiddleware.RequirePermission("read"), appHandler.GetHistory)

		// Actualizar estado (requiere permiso 'update')
		applications.PATCH("/:id/status", authMiddleware.RequirePermission("update"), appHandler.UpdateStatus)
	}

	// Admin routes (solo admins y analysts)
	admin := protected.Group("/admin")
	admin.Use(authMiddleware.RequireRole(entity.RoleAdmin, entity.RoleAnalyst))
	{
		// Stats handler
		statsHandler := handler.NewStatsHandler(db, log)

		// Dashboard stats
		admin.GET("/stats", statsHandler.GetDashboardStats)

		// Country specific stats
		admin.GET("/stats/country/:code", statsHandler.GetCountryStats)

		// Queue stats
		admin.GET("/queue/stats", func(c *gin.Context) {
			stats, err := jobQueue.Stats(c.Request.Context())
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, stats)
		})

		// Recent jobs (para depuración)
		admin.GET("/queue/jobs/recent", func(c *gin.Context) {
			jobs, err := jobQueue.GetRecentJobs(c.Request.Context(), 20)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"jobs":  jobs,
				"count": len(jobs),
			})
		})

		// Failed jobs (para depuración)
		admin.GET("/queue/jobs/failed", func(c *gin.Context) {
			jobs, err := jobQueue.GetFailedJobs(c.Request.Context(), 20)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"jobs":  jobs,
				"count": len(jobs),
			})
		})
	}

	// ==========================================
	// WEBSOCKET
	// ==========================================
	r.GET("/ws", authMiddleware.OptionalAuth(), func(c *gin.Context) {
		websocket.HandleWebSocket(wsHub, c)
	})

	return r
}
