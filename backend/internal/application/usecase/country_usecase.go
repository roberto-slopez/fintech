package usecase

import (
	"context"

	"github.com/fintech-multipass/backend/internal/domain/entity"
	"github.com/fintech-multipass/backend/internal/domain/repository"
	"github.com/fintech-multipass/backend/internal/infrastructure/cache"
	"github.com/fintech-multipass/backend/internal/infrastructure/logger"
	"github.com/google/uuid"
)

// CountryUseCase casos de uso para países
type CountryUseCase struct {
	countryRepo repository.CountryRepository
	cache       cache.CacheService
	log         *logger.Logger
}

// NewCountryUseCase crea una nueva instancia del caso de uso
func NewCountryUseCase(
	countryRepo repository.CountryRepository,
	cache cache.CacheService,
	log *logger.Logger,
) *CountryUseCase {
	return &CountryUseCase{
		countryRepo: countryRepo,
		cache:       cache,
		log:         log,
	}
}

// GetAllCountries obtiene todos los países activos
func (uc *CountryUseCase) GetAllCountries(ctx context.Context, includeInactive bool) ([]entity.Country, error) {
	// Intentar caché si solo queremos activos
	if !includeInactive && uc.cache != nil {
		countries, err := uc.cache.GetAllCountries(ctx)
		if err == nil && len(countries) > 0 {
			return countries, nil
		}
	}

	// Obtener de base de datos
	countries, err := uc.countryRepo.GetAll(ctx, !includeInactive)
	if err != nil {
		return nil, err
	}

	// Cachear si son solo activos
	if !includeInactive && uc.cache != nil {
		_ = uc.cache.SetAllCountries(ctx, countries)
	}

	return countries, nil
}

// GetCountryByCode obtiene un país por código
func (uc *CountryUseCase) GetCountryByCode(ctx context.Context, code string) (*entity.Country, error) {
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

// GetCountryByID obtiene un país por ID
func (uc *CountryUseCase) GetCountryByID(ctx context.Context, id uuid.UUID) (*entity.Country, error) {
	return uc.countryRepo.GetByID(ctx, id)
}

// GetCountryRules obtiene las reglas de un país
func (uc *CountryUseCase) GetCountryRules(ctx context.Context, countryID uuid.UUID) ([]entity.CountryRule, error) {
	return uc.countryRepo.GetRules(ctx, countryID)
}

// GetCountryDocumentTypes obtiene los tipos de documento de un país
func (uc *CountryUseCase) GetCountryDocumentTypes(ctx context.Context, countryID uuid.UUID) ([]entity.DocumentType, error) {
	return uc.countryRepo.GetDocumentTypes(ctx, countryID)
}

// CountryWithDetails país con detalles adicionales
type CountryWithDetails struct {
	entity.Country
	DocumentTypes []entity.DocumentType `json:"document_types"`
	RulesCount    int                   `json:"rules_count"`
}

// GetCountryWithDetails obtiene un país con todos sus detalles
func (uc *CountryUseCase) GetCountryWithDetails(ctx context.Context, code string) (*CountryWithDetails, error) {
	country, err := uc.GetCountryByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	docTypes, err := uc.countryRepo.GetDocumentTypes(ctx, country.ID)
	if err != nil {
		return nil, err
	}

	rules, err := uc.countryRepo.GetRules(ctx, country.ID)
	if err != nil {
		return nil, err
	}

	return &CountryWithDetails{
		Country:       *country,
		DocumentTypes: docTypes,
		RulesCount:    len(rules),
	}, nil
}

