package entity

import (
	"time"

	"github.com/google/uuid"
)

// Job representa un trabajo en la cola para procesamiento asíncrono
type Job struct {
	ID            uuid.UUID  `json:"id"`
	Type          JobType    `json:"type"`
	Status        JobStatus  `json:"status"`
	Priority      int        `json:"priority"`       // Mayor número = mayor prioridad
	Payload       []byte     `json:"payload"`
	Result        []byte     `json:"result,omitempty"`
	ErrorMessage  string     `json:"error_message,omitempty"`
	Attempts      int        `json:"attempts"`
	MaxAttempts   int        `json:"max_attempts"`
	ScheduledAt   time.Time  `json:"scheduled_at"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// JobType tipos de trabajos
type JobType string

const (
	JobTypeRiskEvaluation     JobType = "RISK_EVALUATION"
	JobTypeBankingInfoFetch   JobType = "BANKING_INFO_FETCH"
	JobTypeDocumentValidation JobType = "DOCUMENT_VALIDATION"
	JobTypeNotification       JobType = "NOTIFICATION"
	JobTypeAuditLog           JobType = "AUDIT_LOG"
	JobTypeWebhookCall        JobType = "WEBHOOK_CALL"
	JobTypeStatusUpdate       JobType = "STATUS_UPDATE"
	JobTypeReportGeneration   JobType = "REPORT_GENERATION"
)

// JobStatus estados del trabajo
type JobStatus string

const (
	JobStatusPending    JobStatus = "PENDING"
	JobStatusProcessing JobStatus = "PROCESSING"
	JobStatusCompleted  JobStatus = "COMPLETED"
	JobStatusFailed     JobStatus = "FAILED"
	JobStatusRetrying   JobStatus = "RETRYING"
	JobStatusCancelled  JobStatus = "CANCELLED"
)

// RiskEvaluationPayload payload para evaluación de riesgo
type RiskEvaluationPayload struct {
	ApplicationID uuid.UUID `json:"application_id"`
	CountryID     uuid.UUID `json:"country_id"`
}

// BankingInfoPayload payload para obtener información bancaria
type BankingInfoPayload struct {
	ApplicationID  uuid.UUID `json:"application_id"`
	ProviderID     uuid.UUID `json:"provider_id"`
	DocumentType   string    `json:"document_type"`
	DocumentNumber string    `json:"document_number"`
}

// NotificationPayload payload para notificaciones
type NotificationPayload struct {
	Type          string                 `json:"type"` // EMAIL, SMS, PUSH, WEBHOOK
	Recipient     string                 `json:"recipient"`
	Subject       string                 `json:"subject,omitempty"`
	TemplateID    string                 `json:"template_id"`
	TemplateData  map[string]interface{} `json:"template_data"`
}

// WebhookPayload payload para llamadas webhook
type WebhookPayload struct {
	URL           string                 `json:"url"`
	Method        string                 `json:"method"`
	Headers       map[string]string      `json:"headers,omitempty"`
	Body          map[string]interface{} `json:"body"`
	TimeoutSec    int                    `json:"timeout_sec"`
	ApplicationID *uuid.UUID             `json:"application_id,omitempty"`
}

// AuditLog registro de auditoría
type AuditLog struct {
	ID            uuid.UUID              `json:"id"`
	EntityType    string                 `json:"entity_type"`    // APPLICATION, USER, etc.
	EntityID      uuid.UUID              `json:"entity_id"`
	Action        string                 `json:"action"`         // CREATE, UPDATE, DELETE, STATUS_CHANGE
	ActorType     string                 `json:"actor_type"`     // USER, SYSTEM, WEBHOOK
	ActorID       *uuid.UUID             `json:"actor_id,omitempty"`
	OldValues     map[string]interface{} `json:"old_values,omitempty"`
	NewValues     map[string]interface{} `json:"new_values,omitempty"`
	IPAddress     string                 `json:"ip_address,omitempty"`
	UserAgent     string                 `json:"user_agent,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
}

// WebhookEvent evento de webhook entrante
type WebhookEvent struct {
	ID            uuid.UUID              `json:"id"`
	Source        string                 `json:"source"`         // Identificador del sistema externo
	EventType     string                 `json:"event_type"`
	Payload       map[string]interface{} `json:"payload"`
	Signature     string                 `json:"signature,omitempty"`
	Status        string                 `json:"status"`         // RECEIVED, PROCESSED, FAILED
	ErrorMessage  string                 `json:"error_message,omitempty"`
	ProcessedAt   *time.Time             `json:"processed_at,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
}

