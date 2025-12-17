package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/fintech-multipass/backend/internal/domain/entity"
	"github.com/fintech-multipass/backend/internal/infrastructure/config"
	"github.com/fintech-multipass/backend/internal/infrastructure/database"
	"github.com/fintech-multipass/backend/internal/infrastructure/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// WebhookHandler handler para webhooks entrantes y salientes
type WebhookHandler struct {
	db     *database.PostgresDB
	log    *logger.Logger
	config config.WebhookConfig
}

// NewWebhookHandler crea una nueva instancia del handler
func NewWebhookHandler(db *database.PostgresDB, log *logger.Logger, cfg config.WebhookConfig) *WebhookHandler {
	return &WebhookHandler{
		db:     db,
		log:    log,
		config: cfg,
	}
}

// HandleIncoming maneja webhooks entrantes
// @Summary Recibir webhook
// @Description Recibe eventos de sistemas externos
// @Tags webhooks
// @Accept json
// @Produce json
// @Param source path string true "Identificador del sistema fuente"
// @Param X-Webhook-Signature header string false "Firma HMAC del payload"
// @Success 200 {object} WebhookResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /webhooks/{source} [post]
func (h *WebhookHandler) HandleIncoming(c *gin.Context) {
	source := c.Param("source")

	// Leer body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "read_error",
			Message: "Failed to read request body",
		})
		return
	}

	// Verificar firma si está configurada
	signature := c.GetHeader("X-Webhook-Signature")
	if h.config.Secret != "" {
		if !h.verifySignature(body, signature) {
			h.log.Warn().
				Str("source", source).
				Msg("Invalid webhook signature")
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "invalid_signature",
				Message: "Webhook signature verification failed",
			})
			return
		}
	}

	// Parsear payload
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "parse_error",
			Message: "Failed to parse JSON payload",
		})
		return
	}

	// Obtener tipo de evento
	eventType, _ := payload["event_type"].(string)
	if eventType == "" {
		eventType = "unknown"
	}

	// Crear registro de evento
	event := &entity.WebhookEvent{
		ID:        uuid.New(),
		Source:    source,
		EventType: eventType,
		Payload:   payload,
		Signature: signature,
		Status:    "RECEIVED",
		CreatedAt: time.Now(),
	}

	// Guardar evento en base de datos
	if err := h.saveEvent(c.Request.Context(), event); err != nil {
		h.log.Error().Err(err).Msg("Failed to save webhook event")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "save_error",
			Message: "Failed to save webhook event",
		})
		return
	}

	h.log.Info().
		Str("event_id", event.ID.String()).
		Str("source", source).
		Str("event_type", eventType).
		Msg("Webhook event received")

	// Procesar evento de forma asíncrona
	go h.processEvent(event)

	c.JSON(http.StatusOK, WebhookResponse{
		Success: true,
		EventID: event.ID.String(),
		Message: "Event received and queued for processing",
	})
}

// verifySignature verifica la firma HMAC del webhook
func (h *WebhookHandler) verifySignature(payload []byte, signature string) bool {
	if signature == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(h.config.Secret))
	mac.Write(payload)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSig))
}

// saveEvent guarda un evento de webhook en la base de datos
func (h *WebhookHandler) saveEvent(ctx context.Context, event *entity.WebhookEvent) error {
	payloadJSON, err := json.Marshal(event.Payload)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO webhook_events (id, source, event_type, payload, signature, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	return h.db.Exec(ctx, query,
		event.ID, event.Source, event.EventType, payloadJSON,
		event.Signature, event.Status, event.CreatedAt,
	)
}

// processEvent procesa un evento de webhook
func (h *WebhookHandler) processEvent(event *entity.WebhookEvent) {
	ctx := context.Background()

	h.log.Info().
		Str("event_id", event.ID.String()).
		Str("source", event.Source).
		Msg("Processing webhook event")

	// Procesar según el tipo de evento
	var err error
	switch event.Source {
	case "banking_provider":
		err = h.processBankingProviderEvent(ctx, event)
	case "payment_gateway":
		err = h.processPaymentGatewayEvent(ctx, event)
	default:
		h.log.Warn().
			Str("source", event.Source).
			Msg("Unknown webhook source")
	}

	// Actualizar estado del evento
	status := "PROCESSED"
	var errorMsg string
	if err != nil {
		status = "FAILED"
		errorMsg = err.Error()
		h.log.Error().
			Err(err).
			Str("event_id", event.ID.String()).
			Msg("Failed to process webhook event")
	}

	updateQuery := `
		UPDATE webhook_events 
		SET status = $2, error_message = $3, processed_at = NOW() 
		WHERE id = $1
	`
	_ = h.db.Exec(ctx, updateQuery, event.ID, status, errorMsg)
}

// processBankingProviderEvent procesa eventos de proveedores bancarios
func (h *WebhookHandler) processBankingProviderEvent(ctx context.Context, event *entity.WebhookEvent) error {
	// Obtener application_id del payload
	applicationIDStr, ok := event.Payload["application_id"].(string)
	if !ok {
		return nil // No es un evento relacionado con una aplicación
	}

	applicationID, err := uuid.Parse(applicationIDStr)
	if err != nil {
		return err
	}

	// Procesar según tipo de evento
	switch event.EventType {
	case "credit_report_ready":
		return h.handleCreditReportReady(ctx, applicationID, event.Payload)
	case "verification_complete":
		return h.handleVerificationComplete(ctx, applicationID, event.Payload)
	}

	return nil
}

// processPaymentGatewayEvent procesa eventos del gateway de pagos
func (h *WebhookHandler) processPaymentGatewayEvent(ctx context.Context, event *entity.WebhookEvent) error {
	return nil
}

// handleCreditReportReady maneja el evento de reporte crediticio listo
func (h *WebhookHandler) handleCreditReportReady(ctx context.Context, applicationID uuid.UUID, payload map[string]interface{}) error {
	h.log.Info().
		Str("application_id", applicationID.String()).
		Msg("Handling credit report ready event")

	query := `
		UPDATE credit_applications 
		SET status = 'VALIDATING', updated_at = NOW()
		WHERE id = $1 AND status = 'PENDING_BANK_INFO'
	`
	return h.db.Exec(ctx, query, applicationID)
}

// handleVerificationComplete maneja el evento de verificación completa
func (h *WebhookHandler) handleVerificationComplete(ctx context.Context, applicationID uuid.UUID, payload map[string]interface{}) error {
	h.log.Info().
		Str("application_id", applicationID.String()).
		Msg("Handling verification complete event")

	verified, _ := payload["verified"].(bool)

	if verified {
		query := `
			UPDATE credit_applications 
			SET status = 'VALIDATING', updated_at = NOW()
			WHERE id = $1 AND status = 'PENDING'
		`
		return h.db.Exec(ctx, query, applicationID)
	} else {
		reason, _ := payload["reason"].(string)
		if reason == "" {
			reason = "Document verification failed"
		}
		query := `
			UPDATE credit_applications 
			SET status = 'REJECTED', status_reason = $2, updated_at = NOW()
			WHERE id = $1
		`
		return h.db.Exec(ctx, query, applicationID, reason)
	}
}

// WebhookResponse respuesta de webhook
type WebhookResponse struct {
	Success bool   `json:"success"`
	EventID string `json:"event_id"`
	Message string `json:"message"`
}
