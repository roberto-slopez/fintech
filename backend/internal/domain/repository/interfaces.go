package repository

import (
	"context"

	"github.com/fintech-multipass/backend/internal/domain/entity"
	"github.com/google/uuid"
)

// CountryRepository interface para operaciones con países
type CountryRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Country, error)
	GetByCode(ctx context.Context, code string) (*entity.Country, error)
	GetAll(ctx context.Context, onlyActive bool) ([]entity.Country, error)
	Create(ctx context.Context, country *entity.Country) error
	Update(ctx context.Context, country *entity.Country) error
	GetRules(ctx context.Context, countryID uuid.UUID) ([]entity.CountryRule, error)
	GetDocumentTypes(ctx context.Context, countryID uuid.UUID) ([]entity.DocumentType, error)
}

// CreditApplicationRepository interface para operaciones con solicitudes de crédito
type CreditApplicationRepository interface {
	Create(ctx context.Context, app *entity.CreditApplication) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.CreditApplication, error)
	Update(ctx context.Context, app *entity.CreditApplication) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status entity.ApplicationStatus, reason string) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter entity.ApplicationFilter) (*entity.ApplicationListResult, error)
	GetByDocumentNumber(ctx context.Context, countryID uuid.UUID, documentNumber string) ([]entity.CreditApplication, error)
	
	// Transiciones de estado
	SaveStateTransition(ctx context.Context, transition *entity.StateTransition) error
	GetStateTransitions(ctx context.Context, applicationID uuid.UUID) ([]entity.StateTransition, error)
	
	// Información bancaria
	SaveBankingInfo(ctx context.Context, info *entity.BankingInfo) error
	GetBankingInfo(ctx context.Context, applicationID uuid.UUID) (*entity.BankingInfo, error)
}

// BankingProviderRepository interface para operaciones con proveedores bancarios
type BankingProviderRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entity.BankingProvider, error)
	GetByCountryID(ctx context.Context, countryID uuid.UUID) ([]entity.BankingProvider, error)
	GetActiveByCountry(ctx context.Context, countryID uuid.UUID) (*entity.BankingProvider, error)
	Create(ctx context.Context, provider *entity.BankingProvider) error
	Update(ctx context.Context, provider *entity.BankingProvider) error
	SaveRequest(ctx context.Context, request *entity.BankingRequest) error
}

// UserRepository interface para operaciones con usuarios
type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	Create(ctx context.Context, user *entity.User) error
	Update(ctx context.Context, user *entity.User) error
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, page, pageSize int) ([]entity.User, int64, error)
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
}

// JobRepository interface para operaciones con trabajos en cola
type JobRepository interface {
	Create(ctx context.Context, job *entity.Job) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Job, error)
	GetPending(ctx context.Context, limit int) ([]entity.Job, error)
	GetByType(ctx context.Context, jobType entity.JobType, status entity.JobStatus) ([]entity.Job, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status entity.JobStatus, result []byte, errorMsg string) error
	IncrementAttempts(ctx context.Context, id uuid.UUID) error
	ClaimJob(ctx context.Context, id uuid.UUID, workerID string) error
	GetStats(ctx context.Context) (map[entity.JobStatus]int64, error)
}

// AuditLogRepository interface para operaciones de auditoría
type AuditLogRepository interface {
	Create(ctx context.Context, log *entity.AuditLog) error
	GetByEntityID(ctx context.Context, entityType string, entityID uuid.UUID) ([]entity.AuditLog, error)
	List(ctx context.Context, filter AuditLogFilter) ([]entity.AuditLog, int64, error)
}

// AuditLogFilter filtros para búsqueda de logs de auditoría
type AuditLogFilter struct {
	EntityType *string
	EntityID   *uuid.UUID
	Action     *string
	ActorID    *uuid.UUID
	FromDate   *string
	ToDate     *string
	Page       int
	PageSize   int
}

// WebhookRepository interface para operaciones con webhooks
type WebhookRepository interface {
	SaveEvent(ctx context.Context, event *entity.WebhookEvent) error
	GetEventByID(ctx context.Context, id uuid.UUID) (*entity.WebhookEvent, error)
	UpdateEventStatus(ctx context.Context, id uuid.UUID, status string, errorMsg string) error
	GetPendingEvents(ctx context.Context, limit int) ([]entity.WebhookEvent, error)
}

// Transaction interface para manejo de transacciones
type Transaction interface {
	Begin(ctx context.Context) (context.Context, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

