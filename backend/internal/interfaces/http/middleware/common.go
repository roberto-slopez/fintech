package middleware

import (
	"time"

	"github.com/fintech-multipass/backend/internal/infrastructure/logger"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestLogger middleware para logging de requests
func RequestLogger(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generar request ID
		requestID := uuid.New().String()
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// Tiempo de inicio
		start := time.Now()

		// Procesar request
		c.Next()

		// Calcular duración
		duration := time.Since(start)

		// Log del request
		log.Info().
			Str("request_id", requestID).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Dur("duration", duration).
			Str("client_ip", c.ClientIP()).
			Str("user_agent", c.Request.UserAgent()).
			Msg("HTTP Request")
	}
}

// ErrorHandler middleware para manejo centralizado de errores
func ErrorHandler(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Procesar errores si los hay
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				log.Error().
					Err(err.Err).
					Str("path", c.Request.URL.Path).
					Str("method", c.Request.Method).
					Msg("Request error")
			}
		}
	}
}

// CORSConfig retorna la configuración CORS
func CORSConfig(origins []string) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}

// Recovery middleware para recuperarse de panics
func Recovery(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Error().
					Interface("error", err).
					Str("path", c.Request.URL.Path).
					Str("method", c.Request.Method).
					Msg("Panic recovered")

				c.AbortWithStatusJSON(500, gin.H{
					"error":   "internal_error",
					"message": "An internal error occurred",
				})
			}
		}()
		c.Next()
	}
}

// RateLimiter middleware simple de rate limiting (básico, para producción usar redis)
func RateLimiter(requestsPerMinute int) gin.HandlerFunc {
	// En producción, usar una implementación con Redis para soporte de múltiples instancias
	return func(c *gin.Context) {
		// Implementación básica - en producción usar algo como golang.org/x/time/rate
		c.Next()
	}
}

// SecurityHeaders agrega headers de seguridad
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Next()
	}
}

