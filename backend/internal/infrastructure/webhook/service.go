package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fintech-multipass/backend/internal/domain/entity"
	"github.com/fintech-multipass/backend/internal/infrastructure/database"
	"github.com/fintech-multipass/backend/internal/infrastructure/logger"
	"github.com/google/uuid"
)

// WebhookService servicio para llamadas webhook
type WebhookService struct {
	db         *database.PostgresDB
	log        *logger.Logger
	httpClient *http.Client
	secretKey  string
}

// NewWebhookService crea una nueva instancia del servicio
func NewWebhookService(db *database.PostgresDB, log *logger.Logger, secretKey string) *WebhookService {
	return &WebhookService{
		db:  db,
		log: log,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		secretKey: secretKey,
	}
}

// WebhookEvent evento de webhook
type WebhookEvent struct {
	ID            uuid.UUID              `json:"id"`
	EventType     string                 `json:"event_type"`
	ApplicationID uuid.UUID              `json:"application_id,omitempty"`
	CountryCode   string                 `json:"country_code,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
	Data          map[string]interface{} `json:"data"`
}

// WebhookEndpoint endpoint configurado para webhooks
type WebhookEndpoint struct {
	ID          uuid.UUID `json:"id"`
	CountryID   uuid.UUID `json:"country_id"`
	URL         string    `json:"url"`
	Secret      string    `json:"-"`
	EventTypes  []string  `json:"event_types"`
	IsActive    bool      `json:"is_active"`
	MaxRetries  int       `json:"max_retries"`
	RetryDelay  int       `json:"retry_delay_seconds"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// WebhookDelivery registro de entrega de webhook
type WebhookDelivery struct {
	ID           uuid.UUID  `json:"id"`
	EndpointID   uuid.UUID  `json:"endpoint_id"`
	EventID      uuid.UUID  `json:"event_id"`
	Status       string     `json:"status"` // PENDING, SENT, FAILED
	HTTPStatus   int        `json:"http_status,omitempty"`
	ResponseBody string     `json:"response_body,omitempty"`
	Attempts     int        `json:"attempts"`
	LastAttempt  *time.Time `json:"last_attempt,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// DeliverWebhook envía un webhook a un endpoint
func (s *WebhookService) DeliverWebhook(ctx context.Context, endpoint *WebhookEndpoint, event *WebhookEvent) error {
	// Preparar payload
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook event: %w", err)
	}

	// Crear signature
	signature := s.signPayload(payload, endpoint.Secret)

	// Crear request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.URL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", signature)
	req.Header.Set("X-Webhook-Event", event.EventType)
	req.Header.Set("X-Webhook-ID", event.ID.String())
	req.Header.Set("X-Webhook-Timestamp", event.Timestamp.Format(time.RFC3339))
	req.Header.Set("User-Agent", "Fintech-Multipass-Webhook/1.0")

	// Enviar request
	s.log.Info().
		Str("endpoint", endpoint.URL).
		Str("event_type", event.EventType).
		Str("event_id", event.ID.String()).
		Msg("Sending webhook")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.log.Error().Err(err).Str("endpoint", endpoint.URL).Msg("Webhook delivery failed")
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	// Leer respuesta
	body, _ := io.ReadAll(resp.Body)

	// Verificar status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.log.Warn().
			Int("status", resp.StatusCode).
			Str("body", string(body)).
			Msg("Webhook endpoint returned non-success status")
		return fmt.Errorf("webhook endpoint returned status %d", resp.StatusCode)
	}

	s.log.Info().
		Str("endpoint", endpoint.URL).
		Int("status", resp.StatusCode).
		Msg("Webhook delivered successfully")

	return nil
}

// signPayload firma el payload con HMAC-SHA256
func (s *WebhookService) signPayload(payload []byte, secret string) string {
	if secret == "" {
		secret = s.secretKey
	}
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

// GetEndpointsForEvent obtiene los endpoints configurados para un tipo de evento
func (s *WebhookService) GetEndpointsForEvent(ctx context.Context, countryID uuid.UUID, eventType string) ([]WebhookEndpoint, error) {
	query := `
		SELECT id, country_id, url, secret, event_types, is_active, max_retries, retry_delay_seconds, created_at, updated_at
		FROM webhook_endpoints
		WHERE country_id = $1 AND is_active = true AND $2 = ANY(event_types)
	`

	rows, err := s.db.Query(ctx, query, countryID, eventType)
	if err != nil {
		return nil, fmt.Errorf("failed to query webhook endpoints: %w", err)
	}
	defer rows.Close()

	var endpoints []WebhookEndpoint
	for rows.Next() {
		var ep WebhookEndpoint
		if err := rows.Scan(
			&ep.ID, &ep.CountryID, &ep.URL, &ep.Secret, &ep.EventTypes,
			&ep.IsActive, &ep.MaxRetries, &ep.RetryDelay, &ep.CreatedAt, &ep.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan endpoint: %w", err)
		}
		endpoints = append(endpoints, ep)
	}

	return endpoints, nil
}

// PublishApplicationEvent publica un evento de cambio en una aplicación
func (s *WebhookService) PublishApplicationEvent(ctx context.Context, app *entity.CreditApplication, eventType string) error {
	event := &WebhookEvent{
		ID:            uuid.New(),
		EventType:     eventType,
		ApplicationID: app.ID,
		CountryCode:   "", // Se podría agregar el código del país
		Timestamp:     time.Now(),
		Data: map[string]interface{}{
			"application_id":   app.ID.String(),
			"status":           string(app.Status),
			"status_reason":    app.StatusReason,
			"requested_amount": app.RequestedAmount,
			"requires_review":  app.RequiresReview,
		},
	}

	// Guardar evento
	if err := s.saveEvent(ctx, event); err != nil {
		s.log.Error().Err(err).Msg("Failed to save webhook event")
	}

	// Obtener endpoints
	endpoints, err := s.GetEndpointsForEvent(ctx, app.CountryID, eventType)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to get webhook endpoints")
		return err
	}

	if len(endpoints) == 0 {
		s.log.Debug().Str("event_type", eventType).Msg("No webhook endpoints configured for event")
		return nil
	}

	// Enviar a cada endpoint
	for _, ep := range endpoints {
		if err := s.DeliverWebhook(ctx, &ep, event); err != nil {
			// Log error pero continuar con otros endpoints
			s.log.Error().
				Err(err).
				Str("endpoint_id", ep.ID.String()).
				Msg("Failed to deliver webhook")
		}
	}

	return nil
}

// saveEvent guarda un evento de webhook
func (s *WebhookService) saveEvent(ctx context.Context, event *WebhookEvent) error {
	data, _ := json.Marshal(event.Data)
	query := `
		INSERT INTO webhook_events (id, event_type, application_id, country_code, data, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	return s.db.Exec(ctx, query, event.ID, event.EventType, event.ApplicationID, event.CountryCode, data, event.Timestamp)
}

// WebhookFromJob procesa un job de webhook
func (s *WebhookService) WebhookFromJob(ctx context.Context, job *entity.Job) error {
	var payload struct {
		EndpointID string                 `json:"endpoint_id"`
		Event      map[string]interface{} `json:"event"`
	}

	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("failed to parse webhook job payload: %w", err)
	}

	endpointID, err := uuid.Parse(payload.EndpointID)
	if err != nil {
		return fmt.Errorf("invalid endpoint ID: %w", err)
	}

	// Obtener endpoint
	endpoint, err := s.getEndpointByID(ctx, endpointID)
	if err != nil {
		return fmt.Errorf("failed to get endpoint: %w", err)
	}

	// Crear evento
	event := &WebhookEvent{
		ID:        uuid.New(),
		Timestamp: time.Now(),
		Data:      payload.Event,
	}
	if eventType, ok := payload.Event["event_type"].(string); ok {
		event.EventType = eventType
	}

	return s.DeliverWebhook(ctx, endpoint, event)
}

// getEndpointByID obtiene un endpoint por ID
func (s *WebhookService) getEndpointByID(ctx context.Context, id uuid.UUID) (*WebhookEndpoint, error) {
	query := `
		SELECT id, country_id, url, secret, event_types, is_active, max_retries, retry_delay_seconds, created_at, updated_at
		FROM webhook_endpoints
		WHERE id = $1
	`

	var ep WebhookEndpoint
	row := s.db.QueryRow(ctx, query, id)
	err := row.Scan(
		&ep.ID, &ep.CountryID, &ep.URL, &ep.Secret, &ep.EventTypes,
		&ep.IsActive, &ep.MaxRetries, &ep.RetryDelay, &ep.CreatedAt, &ep.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &ep, nil
}

// Common webhook event types
const (
	EventApplicationCreated   = "application.created"
	EventApplicationUpdated   = "application.updated"
	EventApplicationApproved  = "application.approved"
	EventApplicationRejected  = "application.rejected"
	EventApplicationDisbursed = "application.disbursed"
	EventBankingInfoReceived  = "banking_info.received"
)

