package persistence

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fintech-multipass/backend/internal/domain/entity"
	"github.com/fintech-multipass/backend/internal/infrastructure/database"
	"github.com/google/uuid"
)

// CountryRepository implementación de repositorio de países
type CountryRepository struct {
	db *database.PostgresDB
}

// NewCountryRepository crea una nueva instancia del repositorio
func NewCountryRepository(db *database.PostgresDB) *CountryRepository {
	return &CountryRepository{db: db}
}

// GetByID obtiene un país por ID
func (r *CountryRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Country, error) {
	query := `
		SELECT id, code, name, currency, timezone, is_active, config, created_at, updated_at
		FROM countries
		WHERE id = $1
	`
	
	var country entity.Country
	var configJSON []byte
	
	row := r.db.QueryRow(ctx, query, id)
	err := row.Scan(
		&country.ID, &country.Code, &country.Name, &country.Currency,
		&country.Timezone, &country.IsActive, &configJSON,
		&country.CreatedAt, &country.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("country not found: %w", err)
	}
	
	if err := json.Unmarshal(configJSON, &country.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	return &country, nil
}

// GetByCode obtiene un país por código
func (r *CountryRepository) GetByCode(ctx context.Context, code string) (*entity.Country, error) {
	query := `
		SELECT id, code, name, currency, timezone, is_active, config, created_at, updated_at
		FROM countries
		WHERE code = $1
	`
	
	var country entity.Country
	var configJSON []byte
	
	row := r.db.QueryRow(ctx, query, code)
	err := row.Scan(
		&country.ID, &country.Code, &country.Name, &country.Currency,
		&country.Timezone, &country.IsActive, &configJSON,
		&country.CreatedAt, &country.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("country not found: %w", err)
	}
	
	if err := json.Unmarshal(configJSON, &country.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	return &country, nil
}

// GetAll obtiene todos los países
func (r *CountryRepository) GetAll(ctx context.Context, onlyActive bool) ([]entity.Country, error) {
	query := `
		SELECT id, code, name, currency, timezone, is_active, config, created_at, updated_at
		FROM countries
	`
	if onlyActive {
		query += " WHERE is_active = true"
	}
	query += " ORDER BY name"
	
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query countries: %w", err)
	}
	defer rows.Close()
	
	var countries []entity.Country
	for rows.Next() {
		var country entity.Country
		var configJSON []byte
		
		err := rows.Scan(
			&country.ID, &country.Code, &country.Name, &country.Currency,
			&country.Timezone, &country.IsActive, &configJSON,
			&country.CreatedAt, &country.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan country: %w", err)
		}
		
		if err := json.Unmarshal(configJSON, &country.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}
		
		countries = append(countries, country)
	}
	
	return countries, nil
}

// Create crea un nuevo país
func (r *CountryRepository) Create(ctx context.Context, country *entity.Country) error {
	if country.ID == uuid.Nil {
		country.ID = uuid.New()
	}
	
	configJSON, err := json.Marshal(country.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	query := `
		INSERT INTO countries (id, code, name, currency, timezone, is_active, config)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	
	return r.db.Exec(ctx, query, country.ID, country.Code, country.Name, country.Currency, country.Timezone, country.IsActive, configJSON)
}

// Update actualiza un país
func (r *CountryRepository) Update(ctx context.Context, country *entity.Country) error {
	configJSON, err := json.Marshal(country.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	query := `
		UPDATE countries
		SET code = $2, name = $3, currency = $4, timezone = $5, is_active = $6, config = $7
		WHERE id = $1
	`
	
	return r.db.Exec(ctx, query, country.ID, country.Code, country.Name, country.Currency, country.Timezone, country.IsActive, configJSON)
}

// GetRules obtiene las reglas de un país
func (r *CountryRepository) GetRules(ctx context.Context, countryID uuid.UUID) ([]entity.CountryRule, error) {
	query := `
		SELECT id, country_id, rule_type, name, description, is_active, priority, config, created_at, updated_at
		FROM country_rules
		WHERE country_id = $1 AND is_active = true
		ORDER BY priority DESC
	`
	
	rows, err := r.db.Query(ctx, query, countryID)
	if err != nil {
		return nil, fmt.Errorf("failed to query rules: %w", err)
	}
	defer rows.Close()
	
	var rules []entity.CountryRule
	for rows.Next() {
		var rule entity.CountryRule
		var configJSON []byte
		
		err := rows.Scan(
			&rule.ID, &rule.CountryID, &rule.RuleType, &rule.Name,
			&rule.Description, &rule.IsActive, &rule.Priority, &configJSON,
			&rule.CreatedAt, &rule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}
		
		if err := json.Unmarshal(configJSON, &rule.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}
		
		rules = append(rules, rule)
	}
	
	return rules, nil
}

// GetDocumentTypes obtiene los tipos de documento de un país
func (r *CountryRepository) GetDocumentTypes(ctx context.Context, countryID uuid.UUID) ([]entity.DocumentType, error) {
	query := `
		SELECT id, country_id, code, name, validation_regex, is_required, created_at
		FROM document_types
		WHERE country_id = $1
		ORDER BY is_required DESC, name
	`
	
	rows, err := r.db.Query(ctx, query, countryID)
	if err != nil {
		return nil, fmt.Errorf("failed to query document types: %w", err)
	}
	defer rows.Close()
	
	var docTypes []entity.DocumentType
	for rows.Next() {
		var dt entity.DocumentType
		err := rows.Scan(&dt.ID, &dt.CountryID, &dt.Code, &dt.Name, &dt.ValidationRegex, &dt.IsRequired, &dt.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan document type: %w", err)
		}
		docTypes = append(docTypes, dt)
	}
	
	return docTypes, nil
}

