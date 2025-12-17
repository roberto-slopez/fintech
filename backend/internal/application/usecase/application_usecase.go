package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/fintech-multipass/backend/internal/domain/entity"
	"github.com/fintech-multipass/backend/internal/domain/repository"
	"github.com/fintech-multipass/backend/internal/domain/service"
	"github.com/fintech-multipass/backend/internal/infrastructure/cache"
	"github.com/fintech-multipass/backend/internal/infrastructure/logger"
	"github.com/google/uuid"
)

// ApplicationUseCase casos de uso para solicitudes de crédito
type ApplicationUseCase struct {
	appRepo      repository.CreditApplicationRepository
	countryRepo  repository.CountryRepository
	providerRepo repository.BankingProviderRepository
	validator    service.RuleValidator
	cache        cache.CacheService
	eventPub     service.EventPublisher
	jobQueue     service.JobQueue
	log          *logger.Logger
}

// NewApplicationUseCase crea una nueva instancia del caso de uso
func NewApplicationUseCase(
	appRepo repository.CreditApplicationRepository,
	countryRepo repository.CountryRepository,
	providerRepo repository.BankingProviderRepository,
	validator service.RuleValidator,
	cache cache.CacheService,
	eventPub service.EventPublisher,
	jobQueue service.JobQueue,
	log *logger.Logger,
) *ApplicationUseCase {
	return &ApplicationUseCase{
		appRepo:      appRepo,
		countryRepo:  countryRepo,
		providerRepo: providerRepo,
		validator:    validator,
		cache:        cache,
		eventPub:     eventPub,
		jobQueue:     jobQueue,
		log:          log,
	}
}

// CreateApplicationInput datos de entrada para crear una solicitud
type CreateApplicationInput struct {
	CountryCode     string  `json:"country_code" binding:"required"`
	FullName        string  `json:"full_name" binding:"required,min=3,max=200"`
	DocumentType    string  `json:"document_type" binding:"required"`
	DocumentNumber  string  `json:"document_number" binding:"required"`
	Email           string  `json:"email" binding:"required,email"`
	Phone           string  `json:"phone,omitempty"`
	RequestedAmount float64 `json:"requested_amount" binding:"required,gt=0"`
	MonthlyIncome   float64 `json:"monthly_income" binding:"required,gt=0"`
	IPAddress       string  `json:"-"`
	UserAgent       string  `json:"-"`
}

// CreateApplication crea una nueva solicitud de crédito
func (uc *ApplicationUseCase) CreateApplication(ctx context.Context, input CreateApplicationInput) (*entity.CreditApplication, error) {
	uc.log.Info().
		Str("country", input.CountryCode).
		Str("document", input.DocumentNumber).
		Float64("amount", input.RequestedAmount).
		Msg("Creating credit application")

	// 1. Obtener país
	country, err := uc.getCountryByCode(ctx, input.CountryCode)
	if err != nil {
		return nil, fmt.Errorf("invalid country: %w", err)
	}

	// 2. Validar montos contra configuración del país
	if err := uc.validateAmountLimits(input.RequestedAmount, country.Config); err != nil {
		return nil, err
	}

	// 3. Validar tipo de documento
	docTypes, err := uc.countryRepo.GetDocumentTypes(ctx, country.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get document types: %w", err)
	}
	if !uc.isValidDocumentType(input.DocumentType, docTypes) {
		return nil, fmt.Errorf("invalid document type '%s' for country %s", input.DocumentType, input.CountryCode)
	}

	// 4. Crear entidad de solicitud
	app := &entity.CreditApplication{
		ID:              uuid.New(),
		CountryID:       country.ID,
		Country:         country,
		FullName:        input.FullName,
		DocumentType:    input.DocumentType,
		DocumentNumber:  input.DocumentNumber,
		Email:           input.Email,
		Phone:           input.Phone,
		RequestedAmount: input.RequestedAmount,
		MonthlyIncome:   input.MonthlyIncome,
		Status:          entity.StatusPending,
		RequiresReview:  input.RequestedAmount >= country.Config.ReviewThreshold,
		ApplicationDate: time.Now(),
		CreatedByIP:     input.IPAddress,
		UserAgent:       input.UserAgent,
	}

	// 5. Guardar en base de datos
	// El trigger de PostgreSQL creará automáticamente los jobs de validación y obtención de info bancaria
	if err := uc.appRepo.Create(ctx, app); err != nil {
		return nil, fmt.Errorf("failed to create application: %w", err)
	}

	// 6. Publicar evento para WebSocket
	if uc.eventPub != nil {
		_ = uc.eventPub.PublishNewApplication(ctx, app)
	}

	// 7. Cachear aplicación
	if uc.cache != nil {
		_ = uc.cache.SetApplication(ctx, app)
	}

	uc.log.Info().
		Str("application_id", app.ID.String()).
		Str("status", string(app.Status)).
		Bool("requires_review", app.RequiresReview).
		Msg("Credit application created")

	return app, nil
}

// GetApplication obtiene una solicitud por ID
func (uc *ApplicationUseCase) GetApplication(ctx context.Context, id uuid.UUID) (*entity.CreditApplication, error) {
	// Intentar obtener de caché primero
	if uc.cache != nil {
		app, err := uc.cache.GetApplication(ctx, id)
		if err == nil && app != nil {
			return app, nil
		}
	}

	// Obtener de base de datos
	app, err := uc.appRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("application not found: %w", err)
	}

	// Cachear para futuras consultas
	if uc.cache != nil {
		_ = uc.cache.SetApplication(ctx, app)
	}

	return app, nil
}

// ListApplications lista solicitudes con filtros
func (uc *ApplicationUseCase) ListApplications(ctx context.Context, filter entity.ApplicationFilter) (*entity.ApplicationListResult, error) {
	return uc.appRepo.List(ctx, filter)
}

// UpdateStatusInput datos de entrada para actualizar estado
type UpdateStatusInput struct {
	ApplicationID uuid.UUID `json:"-"`
	NewStatus     entity.ApplicationStatus `json:"status" binding:"required"`
	Reason        string `json:"reason,omitempty"`
	TriggeredBy   string `json:"-"` // USER, SYSTEM, WEBHOOK
	TriggeredByID *uuid.UUID `json:"-"`
}

// UpdateStatus actualiza el estado de una solicitud
func (uc *ApplicationUseCase) UpdateStatus(ctx context.Context, input UpdateStatusInput) (*entity.CreditApplication, error) {
	// 1. Obtener aplicación actual
	app, err := uc.appRepo.GetByID(ctx, input.ApplicationID)
	if err != nil {
		return nil, fmt.Errorf("application not found: %w", err)
	}

	// 2. Verificar transición válida
	if !app.Status.CanTransitionTo(input.NewStatus) {
		return nil, fmt.Errorf("invalid status transition from %s to %s", app.Status, input.NewStatus)
	}

	oldStatus := app.Status

	// 3. Actualizar estado
	if err := uc.appRepo.UpdateStatus(ctx, input.ApplicationID, input.NewStatus, input.Reason); err != nil {
		return nil, fmt.Errorf("failed to update status: %w", err)
	}

	// 4. Guardar transición de estado
	transition := &entity.StateTransition{
		ApplicationID: input.ApplicationID,
		FromStatus:    oldStatus,
		ToStatus:      input.NewStatus,
		Reason:        input.Reason,
		TriggeredBy:   input.TriggeredBy,
		TriggeredByID: input.TriggeredByID,
	}
	_ = uc.appRepo.SaveStateTransition(ctx, transition)

	// 5. Obtener aplicación actualizada
	app, _ = uc.appRepo.GetByID(ctx, input.ApplicationID)

	// 6. Invalidar caché
	if uc.cache != nil {
		_ = uc.cache.InvalidateApplication(ctx, input.ApplicationID)
	}

	// 7. Publicar evento para WebSocket
	if uc.eventPub != nil {
		_ = uc.eventPub.PublishStatusChange(ctx, input.ApplicationID, oldStatus, input.NewStatus)
	}

	uc.log.Info().
		Str("application_id", input.ApplicationID.String()).
		Str("from_status", string(oldStatus)).
		Str("to_status", string(input.NewStatus)).
		Msg("Application status updated")

	return app, nil
}

// GetApplicationHistory obtiene el historial de transiciones de una solicitud
func (uc *ApplicationUseCase) GetApplicationHistory(ctx context.Context, id uuid.UUID) ([]entity.StateTransition, error) {
	return uc.appRepo.GetStateTransitions(ctx, id)
}

// Helper methods

func (uc *ApplicationUseCase) getCountryByCode(ctx context.Context, code string) (*entity.Country, error) {
	// Intentar caché primero
	if uc.cache != nil {
		country, err := uc.cache.GetCountry(ctx, code)
		if err == nil && country != nil {
			return country, nil
		}
	}

	// Obtener de base de datos
	country, err := uc.countryRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	// Cachear
	if uc.cache != nil {
		_ = uc.cache.SetCountry(ctx, country)
	}

	return country, nil
}

func (uc *ApplicationUseCase) validateAmountLimits(amount float64, config entity.CountryConfig) error {
	if amount < config.MinLoanAmount {
		return fmt.Errorf("requested amount %.2f is below minimum %.2f", amount, config.MinLoanAmount)
	}
	if amount > config.MaxLoanAmount {
		return fmt.Errorf("requested amount %.2f exceeds maximum %.2f", amount, config.MaxLoanAmount)
	}
	return nil
}

func (uc *ApplicationUseCase) isValidDocumentType(docType string, validTypes []entity.DocumentType) bool {
	for _, dt := range validTypes {
		if dt.Code == docType {
			return true
		}
	}
	return false
}

