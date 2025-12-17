package service

import (
	"context"

	"github.com/fintech-multipass/backend/internal/domain/entity"
	"github.com/google/uuid"
)

// RuleValidator interface para validación de reglas por país
type RuleValidator interface {
	// ValidateApplication valida una solicitud según las reglas del país
	ValidateApplication(ctx context.Context, app *entity.CreditApplication, rules []entity.CountryRule) ([]entity.ValidationResult, error)
	
	// ValidateDocument valida un documento según el tipo y país
	ValidateDocument(ctx context.Context, docType, docNumber, countryCode string) (bool, string, error)
}

// BankingService interface para integración con proveedores bancarios
type BankingService interface {
	// FetchBankingInfo obtiene información bancaria del proveedor correspondiente al país
	FetchBankingInfo(ctx context.Context, provider *entity.BankingProvider, docType, docNumber string) (*entity.BankingInfoResponse, error)
	
	// GetProviderForCountry obtiene el proveedor activo para un país
	GetProviderForCountry(ctx context.Context, countryID uuid.UUID) (*entity.BankingProvider, error)
}

// RiskEvaluator interface para evaluación de riesgo
type RiskEvaluator interface {
	// EvaluateRisk evalúa el riesgo de una solicitud
	EvaluateRisk(ctx context.Context, app *entity.CreditApplication, bankingInfo *entity.BankingInfo) (float64, []string, error)
	
	// ShouldRequireReview determina si la solicitud requiere revisión manual
	ShouldRequireReview(ctx context.Context, app *entity.CreditApplication, riskScore float64) bool
}

// NotificationService interface para envío de notificaciones
type NotificationService interface {
	// SendStatusNotification envía notificación de cambio de estado
	SendStatusNotification(ctx context.Context, app *entity.CreditApplication, newStatus entity.ApplicationStatus) error
	
	// SendWebhook envía webhook a sistema externo
	SendWebhook(ctx context.Context, payload *entity.WebhookPayload) error
}

// CacheService interface para operaciones de caché
type CacheService interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttlSeconds int) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	
	// Keys específicas
	GetApplication(ctx context.Context, id uuid.UUID) (*entity.CreditApplication, error)
	SetApplication(ctx context.Context, app *entity.CreditApplication) error
	InvalidateApplication(ctx context.Context, id uuid.UUID) error
	
	GetCountry(ctx context.Context, code string) (*entity.Country, error)
	SetCountry(ctx context.Context, country *entity.Country) error
	GetAllCountries(ctx context.Context) ([]entity.Country, error)
	SetAllCountries(ctx context.Context, countries []entity.Country) error
}

// JobQueue interface para cola de trabajos
type JobQueue interface {
	// Enqueue agrega un trabajo a la cola
	Enqueue(ctx context.Context, job *entity.Job) error
	
	// EnqueueWithDelay agrega un trabajo con retraso
	EnqueueWithDelay(ctx context.Context, job *entity.Job, delaySec int) error
	
	// Dequeue obtiene el siguiente trabajo pendiente
	Dequeue(ctx context.Context) (*entity.Job, error)
	
	// Complete marca un trabajo como completado
	Complete(ctx context.Context, jobID uuid.UUID, result []byte) error
	
	// Fail marca un trabajo como fallido
	Fail(ctx context.Context, jobID uuid.UUID, errorMsg string) error
	
	// Retry reencola un trabajo para reintento
	Retry(ctx context.Context, jobID uuid.UUID) error
	
	// StartWorkers inicia los workers de procesamiento
	StartWorkers(ctx context.Context, count int)
	
	// Stats obtiene estadísticas de la cola
	Stats(ctx context.Context) (map[entity.JobStatus]int64, error)
}

// EventPublisher interface para publicar eventos en tiempo real
type EventPublisher interface {
	// PublishApplicationUpdate publica actualización de solicitud
	PublishApplicationUpdate(ctx context.Context, app *entity.CreditApplication) error
	
	// PublishStatusChange publica cambio de estado
	PublishStatusChange(ctx context.Context, applicationID uuid.UUID, oldStatus, newStatus entity.ApplicationStatus) error
	
	// PublishNewApplication publica nueva solicitud creada
	PublishNewApplication(ctx context.Context, app *entity.CreditApplication) error
	
	// Subscribe suscribe a eventos de un tipo específico
	Subscribe(ctx context.Context, eventType string, handler func(data interface{})) error
}

// AuditService interface para registro de auditoría
type AuditService interface {
	// LogAction registra una acción de auditoría
	LogAction(ctx context.Context, log *entity.AuditLog) error
	
	// LogApplicationChange registra un cambio en una solicitud
	LogApplicationChange(ctx context.Context, app *entity.CreditApplication, action string, actorType string, actorID *uuid.UUID, oldValues, newValues map[string]interface{}) error
}

// TokenService interface para manejo de tokens JWT
type TokenService interface {
	// GenerateToken genera un token de acceso
	GenerateToken(user *entity.User) (string, error)
	
	// GenerateRefreshToken genera un token de refresco
	GenerateRefreshToken(user *entity.User) (string, error)
	
	// ValidateToken valida un token y retorna los claims
	ValidateToken(token string) (*entity.TokenClaims, error)
	
	// RefreshToken refresca un token expirado
	RefreshToken(refreshToken string) (string, error)
}

// PasswordService interface para manejo de contraseñas
type PasswordService interface {
	Hash(password string) (string, error)
	Verify(password, hash string) bool
}

