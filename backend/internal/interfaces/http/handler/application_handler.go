package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/fintech-multipass/backend/internal/application/usecase"
	"github.com/fintech-multipass/backend/internal/domain/entity"
	"github.com/fintech-multipass/backend/internal/infrastructure/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ApplicationHandler handler para solicitudes de crédito
type ApplicationHandler struct {
	usecase *usecase.ApplicationUseCase
	log     *logger.Logger
}

// NewApplicationHandler crea una nueva instancia del handler
func NewApplicationHandler(uc *usecase.ApplicationUseCase, log *logger.Logger) *ApplicationHandler {
	return &ApplicationHandler{
		usecase: uc,
		log:     log,
	}
}

// Create crea una nueva solicitud de crédito
// @Summary Crear solicitud de crédito
// @Description Crea una nueva solicitud de crédito para un país específico
// @Tags applications
// @Accept json
// @Produce json
// @Param input body usecase.CreateApplicationInput true "Datos de la solicitud"
// @Success 201 {object} entity.CreditApplication
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /applications [post]
func (h *ApplicationHandler) Create(c *gin.Context) {
	var input usecase.CreateApplicationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
		return
	}

	// Obtener IP y User-Agent
	input.IPAddress = c.ClientIP()
	input.UserAgent = c.GetHeader("User-Agent")

	app, err := h.usecase.CreateApplication(c.Request.Context(), input)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to create application")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "create_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, app)
}

// GetByID obtiene una solicitud por ID
// @Summary Obtener solicitud
// @Description Obtiene los detalles de una solicitud específica
// @Tags applications
// @Produce json
// @Param id path string true "ID de la solicitud"
// @Success 200 {object} entity.CreditApplication
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /applications/{id} [get]
func (h *ApplicationHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid application ID format",
		})
		return
	}

	app, err := h.usecase.GetApplication(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Application not found",
		})
		return
	}

	c.JSON(http.StatusOK, app)
}

// List lista solicitudes con filtros
// @Summary Listar solicitudes
// @Description Obtiene una lista paginada de solicitudes con filtros opcionales
// @Tags applications
// @Produce json
// @Param country query string false "Código de país (ES, MX, CO, etc.)"
// @Param status query string false "Estado de la solicitud"
// @Param requires_review query boolean false "Solo solicitudes que requieren revisión"
// @Param from_date query string false "Fecha desde (RFC3339)"
// @Param to_date query string false "Fecha hasta (RFC3339)"
// @Param min_amount query number false "Monto mínimo"
// @Param max_amount query number false "Monto máximo"
// @Param search query string false "Búsqueda por nombre o documento"
// @Param page query int false "Número de página" default(1)
// @Param page_size query int false "Tamaño de página" default(20)
// @Param sort_by query string false "Campo de ordenamiento"
// @Param sort_order query string false "Orden (ASC/DESC)" default(DESC)
// @Success 200 {object} entity.ApplicationListResult
// @Failure 400 {object} ErrorResponse
// @Security BearerAuth
// @Router /applications [get]
func (h *ApplicationHandler) List(c *gin.Context) {
	filter := entity.ApplicationFilter{
		Page:      1,
		PageSize:  20,
		SortOrder: "DESC",
	}

	// Parsear parámetros de query
	if countryCode := c.Query("country"); countryCode != "" {
		filter.CountryCode = &countryCode
	}

	if status := c.Query("status"); status != "" {
		s := entity.ApplicationStatus(status)
		filter.Status = &s
	}

	if requiresReview := c.Query("requires_review"); requiresReview != "" {
		r := requiresReview == "true"
		filter.RequiresReview = &r
	}

	if fromDate := c.Query("from_date"); fromDate != "" {
		if t, err := time.Parse(time.RFC3339, fromDate); err == nil {
			filter.FromDate = &t
		}
	}

	if toDate := c.Query("to_date"); toDate != "" {
		if t, err := time.Parse(time.RFC3339, toDate); err == nil {
			filter.ToDate = &t
		}
	}

	if minAmount := c.Query("min_amount"); minAmount != "" {
		if a, err := strconv.ParseFloat(minAmount, 64); err == nil {
			filter.MinAmount = &a
		}
	}

	if maxAmount := c.Query("max_amount"); maxAmount != "" {
		if a, err := strconv.ParseFloat(maxAmount, 64); err == nil {
			filter.MaxAmount = &a
		}
	}

	if search := c.Query("search"); search != "" {
		filter.SearchTerm = &search
	}

	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			filter.Page = p
		}
	}

	if pageSize := c.Query("page_size"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil {
			filter.PageSize = ps
		}
	}

	if sortBy := c.Query("sort_by"); sortBy != "" {
		filter.SortBy = sortBy
	}

	if sortOrder := c.Query("sort_order"); sortOrder != "" {
		filter.SortOrder = sortOrder
	}

	result, err := h.usecase.ListApplications(c.Request.Context(), filter)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to list applications")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "list_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// UpdateStatus actualiza el estado de una solicitud
// @Summary Actualizar estado
// @Description Actualiza el estado de una solicitud específica
// @Tags applications
// @Accept json
// @Produce json
// @Param id path string true "ID de la solicitud"
// @Param input body UpdateStatusRequest true "Nuevo estado"
// @Success 200 {object} entity.CreditApplication
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /applications/{id}/status [patch]
func (h *ApplicationHandler) UpdateStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid application ID format",
		})
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
		return
	}

	// Obtener usuario del contexto (establecido por middleware de autenticación)
	userID, _ := c.Get("user_id")
	var triggeredByID *uuid.UUID
	if uid, ok := userID.(uuid.UUID); ok {
		triggeredByID = &uid
	}

	input := usecase.UpdateStatusInput{
		ApplicationID: id,
		NewStatus:     entity.ApplicationStatus(req.Status),
		Reason:        req.Reason,
		TriggeredBy:   "USER",
		TriggeredByID: triggeredByID,
	}

	app, err := h.usecase.UpdateStatus(c.Request.Context(), input)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "application not found" {
			status = http.StatusNotFound
		}
		c.JSON(status, ErrorResponse{
			Error:   "update_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, app)
}

// GetHistory obtiene el historial de transiciones de una solicitud
// @Summary Obtener historial
// @Description Obtiene el historial de cambios de estado de una solicitud
// @Tags applications
// @Produce json
// @Param id path string true "ID de la solicitud"
// @Success 200 {array} entity.StateTransition
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /applications/{id}/history [get]
func (h *ApplicationHandler) GetHistory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid application ID format",
		})
		return
	}

	history, err := h.usecase.GetApplicationHistory(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "fetch_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, history)
}

// UpdateStatusRequest request para actualizar estado
type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"`
	Reason string `json:"reason,omitempty"`
}

// ErrorResponse respuesta de error
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

