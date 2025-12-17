package middleware

import (
	"net/http"
	"strings"

	"github.com/fintech-multipass/backend/internal/application/usecase"
	"github.com/fintech-multipass/backend/internal/domain/entity"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthMiddleware middleware de autenticación JWT
type AuthMiddleware struct {
	authUseCase *usecase.AuthUseCase
}

// NewAuthMiddleware crea una nueva instancia del middleware
func NewAuthMiddleware(authUseCase *usecase.AuthUseCase) *AuthMiddleware {
	return &AuthMiddleware{
		authUseCase: authUseCase,
	}
}

// Authenticate middleware que verifica el token JWT
func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Authorization header is required",
			})
			return
		}

		// Extraer token del header "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid authorization header format",
			})
			return
		}

		token := parts[1]

		// Validar token
		claims, err := m.authUseCase.ValidateToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid or expired token",
			})
			return
		}

		// Establecer claims en el contexto
		c.Set("claims", claims)
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)
		c.Set("country_ids", claims.CountryIDs)

		c.Next()
	}
}

// RequireRole middleware que verifica el rol del usuario
func (m *AuthMiddleware) RequireRole(roles ...entity.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("user_role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "No role information found",
			})
			return
		}

		userRole, ok := roleVal.(entity.UserRole)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid role information",
			})
			return
		}

		// Verificar si el usuario tiene alguno de los roles permitidos
		allowed := false
		for _, role := range roles {
			if userRole == role {
				allowed = true
				break
			}
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "Insufficient permissions",
			})
			return
		}

		c.Next()
	}
}

// RequirePermission middleware que verifica un permiso específico
func (m *AuthMiddleware) RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("user_role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "No role information found",
			})
			return
		}

		userRole, ok := roleVal.(entity.UserRole)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid role information",
			})
			return
		}

		// Verificar permiso basado en el rol
		user := &entity.User{Role: userRole}
		if !user.HasPermission(permission) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "Insufficient permissions for this action",
			})
			return
		}

		c.Next()
	}
}

// RequireCountryAccess middleware que verifica acceso a un país específico
func (m *AuthMiddleware) RequireCountryAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, _ := c.Get("user_role")
		userRole, _ := roleVal.(entity.UserRole)

		// Los admins tienen acceso a todo
		if userRole == entity.RoleAdmin {
			c.Next()
			return
		}

		countryIDsVal, exists := c.Get("country_ids")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "No country access configured",
			})
			return
		}

		countryIDs, ok := countryIDsVal.([]uuid.UUID)
		if !ok || len(countryIDs) == 0 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "No country access configured",
			})
			return
		}

		// Obtener country_id del request (desde body, query o path)
		var requestCountryID uuid.UUID

		// Intentar obtener de query param
		if countryIDStr := c.Query("country_id"); countryIDStr != "" {
			if id, err := uuid.Parse(countryIDStr); err == nil {
				requestCountryID = id
			}
		}

		// Si no hay country_id en la request, permitir (se filtrará después)
		if requestCountryID == uuid.Nil {
			c.Next()
			return
		}

		// Verificar si el usuario tiene acceso al país
		hasAccess := false
		for _, id := range countryIDs {
			if id == requestCountryID {
				hasAccess = true
				break
			}
		}

		if !hasAccess {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "No access to this country",
			})
			return
		}

		c.Next()
	}
}

// OptionalAuth middleware que intenta autenticar pero no falla si no hay token
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		token := parts[1]
		claims, err := m.authUseCase.ValidateToken(token)
		if err != nil {
			c.Next()
			return
		}

		c.Set("claims", claims)
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)
		c.Set("country_ids", claims.CountryIDs)

		c.Next()
	}
}

