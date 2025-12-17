package handler

import (
	"net/http"

	"github.com/fintech-multipass/backend/internal/application/usecase"
	"github.com/fintech-multipass/backend/internal/infrastructure/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CountryHandler handler para países
type CountryHandler struct {
	usecase *usecase.CountryUseCase
	log     *logger.Logger
}

// NewCountryHandler crea una nueva instancia del handler
func NewCountryHandler(uc *usecase.CountryUseCase, log *logger.Logger) *CountryHandler {
	return &CountryHandler{
		usecase: uc,
		log:     log,
	}
}

// GetAll obtiene todos los países
// @Summary Listar países
// @Description Obtiene la lista de todos los países configurados
// @Tags countries
// @Produce json
// @Param include_inactive query boolean false "Incluir países inactivos"
// @Success 200 {array} entity.Country
// @Failure 500 {object} ErrorResponse
// @Router /countries [get]
func (h *CountryHandler) GetAll(c *gin.Context) {
	includeInactive := c.Query("include_inactive") == "true"

	countries, err := h.usecase.GetAllCountries(c.Request.Context(), includeInactive)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to get countries")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "fetch_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, countries)
}

// GetByCode obtiene un país por código
// @Summary Obtener país
// @Description Obtiene los detalles de un país específico
// @Tags countries
// @Produce json
// @Param code path string true "Código del país (ES, MX, CO, etc.)"
// @Success 200 {object} usecase.CountryWithDetails
// @Failure 404 {object} ErrorResponse
// @Router /countries/{code} [get]
func (h *CountryHandler) GetByCode(c *gin.Context) {
	code := c.Param("code")

	country, err := h.usecase.GetCountryWithDetails(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Country not found",
		})
		return
	}

	c.JSON(http.StatusOK, country)
}

// GetRules obtiene las reglas de un país
// @Summary Obtener reglas de país
// @Description Obtiene las reglas de validación configuradas para un país
// @Tags countries
// @Produce json
// @Param code path string true "Código del país"
// @Success 200 {array} entity.CountryRule
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /countries/{code}/rules [get]
func (h *CountryHandler) GetRules(c *gin.Context) {
	code := c.Param("code")

	country, err := h.usecase.GetCountryByCode(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Country not found",
		})
		return
	}

	rules, err := h.usecase.GetCountryRules(c.Request.Context(), country.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "fetch_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, rules)
}

// GetDocumentTypes obtiene los tipos de documento de un país
// @Summary Obtener tipos de documento
// @Description Obtiene los tipos de documento válidos para un país
// @Tags countries
// @Produce json
// @Param code path string true "Código del país"
// @Success 200 {array} entity.DocumentType
// @Failure 404 {object} ErrorResponse
// @Router /countries/{code}/document-types [get]
func (h *CountryHandler) GetDocumentTypes(c *gin.Context) {
	code := c.Param("code")

	country, err := h.usecase.GetCountryByCode(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Country not found",
		})
		return
	}

	docTypes, err := h.usecase.GetCountryDocumentTypes(c.Request.Context(), country.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "fetch_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, docTypes)
}

// Placeholder para evitar error de importación no usada
var _ = uuid.New

