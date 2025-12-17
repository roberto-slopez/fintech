package handler

import (
	"net/http"

	"github.com/fintech-multipass/backend/internal/application/usecase"
	"github.com/fintech-multipass/backend/internal/infrastructure/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthHandler handler para autenticación
type AuthHandler struct {
	usecase *usecase.AuthUseCase
	log     *logger.Logger
}

// NewAuthHandler crea una nueva instancia del handler
func NewAuthHandler(uc *usecase.AuthUseCase, log *logger.Logger) *AuthHandler {
	return &AuthHandler{
		usecase: uc,
		log:     log,
	}
}

// Login autentica a un usuario
// @Summary Iniciar sesión
// @Description Autentica a un usuario y retorna tokens JWT
// @Tags auth
// @Accept json
// @Produce json
// @Param input body usecase.LoginInput true "Credenciales"
// @Success 200 {object} usecase.LoginOutput
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var input usecase.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
		return
	}

	result, err := h.usecase.Login(c.Request.Context(), input)
	if err != nil {
		h.log.Warn().Str("email", input.Email).Err(err).Msg("Login failed")
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Register registra un nuevo usuario
// @Summary Registrar usuario
// @Description Registra un nuevo usuario en el sistema
// @Tags auth
// @Accept json
// @Produce json
// @Param input body usecase.RegisterInput true "Datos del usuario"
// @Success 201 {object} entity.User
// @Failure 400 {object} ErrorResponse
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var input usecase.RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
		return
	}

	user, err := h.usecase.Register(c.Request.Context(), input)
	if err != nil {
		h.log.Error().Err(err).Msg("Registration failed")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "registration_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// RefreshToken refresca el token de acceso
// @Summary Refrescar token
// @Description Obtiene un nuevo token de acceso usando el refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param input body RefreshTokenRequest true "Refresh token"
// @Success 200 {object} usecase.LoginOutput
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
		return
	}

	result, err := h.usecase.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Me obtiene el usuario actual
// @Summary Obtener usuario actual
// @Description Obtiene los datos del usuario autenticado
// @Tags auth
// @Produce json
// @Success 200 {object} entity.User
// @Failure 401 {object} ErrorResponse
// @Security BearerAuth
// @Router /auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "No authentication found",
		})
		return
	}

	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid user ID",
		})
		return
	}

	user, err := h.usecase.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, user)
}

// RefreshTokenRequest request para refrescar token
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
