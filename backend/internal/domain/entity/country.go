package entity

import (
	"time"

	"github.com/google/uuid"
)

// Country representa un país donde opera la fintech
// El sistema está diseñado para soportar N países de forma configurable
type Country struct {
	ID           uuid.UUID       `json:"id"`
	Code         string          `json:"code"`          // ES, PT, IT, MX, CO, BR, etc.
	Name         string          `json:"name"`          // España, Portugal, etc.
	Currency     string          `json:"currency"`      // EUR, MXN, COP, BRL, etc.
	Timezone     string          `json:"timezone"`      // Europe/Madrid, America/Mexico_City, etc.
	IsActive     bool            `json:"is_active"`
	Config       CountryConfig   `json:"config"`        // Configuración específica del país
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// CountryConfig contiene la configuración específica del país
type CountryConfig struct {
	MinLoanAmount        float64 `json:"min_loan_amount"`
	MaxLoanAmount        float64 `json:"max_loan_amount"`
	MinIncomeRequired    float64 `json:"min_income_required"`
	MaxDebtToIncomeRatio float64 `json:"max_debt_to_income_ratio"`
	ReviewThreshold      float64 `json:"review_threshold"`       // Monto a partir del cual requiere revisión
	MinCreditScore       int     `json:"min_credit_score"`
}

// DocumentType representa los tipos de documentos válidos por país
type DocumentType struct {
	ID              uuid.UUID `json:"id"`
	CountryID       uuid.UUID `json:"country_id"`
	Code            string    `json:"code"`             // DNI, NIF, CURP, CPF, CC, CF
	Name            string    `json:"name"`             // Nombre completo del documento
	ValidationRegex string    `json:"validation_regex"` // Regex para validación
	IsRequired      bool      `json:"is_required"`
	CreatedAt       time.Time `json:"created_at"`
}

// CountryRule representa una regla de validación configurable por país
type CountryRule struct {
	ID          uuid.UUID              `json:"id"`
	CountryID   uuid.UUID              `json:"country_id"`
	RuleType    RuleType               `json:"rule_type"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	IsActive    bool                   `json:"is_active"`
	Priority    int                    `json:"priority"`   // Orden de evaluación
	Config      map[string]interface{} `json:"config"`     // Configuración flexible de la regla
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// RuleType tipos de reglas de validación
type RuleType string

const (
	RuleTypeDocumentValidation RuleType = "DOCUMENT_VALIDATION"
	RuleTypeIncomeCheck        RuleType = "INCOME_CHECK"
	RuleTypeDebtRatio          RuleType = "DEBT_RATIO"
	RuleTypeCreditScore        RuleType = "CREDIT_SCORE"
	RuleTypeAmountThreshold    RuleType = "AMOUNT_THRESHOLD"
	RuleTypeCustom             RuleType = "CUSTOM"
)

// ValidationResult resultado de aplicar una regla
type ValidationResult struct {
	RuleID      uuid.UUID `json:"rule_id"`
	RuleName    string    `json:"rule_name"`
	Passed      bool      `json:"passed"`
	Message     string    `json:"message"`
	RequiresReview bool   `json:"requires_review"`
}

