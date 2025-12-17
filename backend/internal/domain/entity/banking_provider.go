package entity

import (
	"time"

	"github.com/google/uuid"
)

// BankingProvider representa un proveedor bancario por país
type BankingProvider struct {
	ID           uuid.UUID              `json:"id"`
	CountryID    uuid.UUID              `json:"country_id"`
	Code         string                 `json:"code"`         // Identificador único del proveedor
	Name         string                 `json:"name"`
	Type         ProviderType           `json:"type"`
	IsActive     bool                   `json:"is_active"`
	Priority     int                    `json:"priority"`     // Orden de preferencia si hay múltiples
	Config       ProviderConfig         `json:"config"`
	Credentials  map[string]string      `json:"-"`            // No exponer credenciales
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// ProviderType tipos de proveedores bancarios
type ProviderType string

const (
	ProviderTypeCreditBureau  ProviderType = "CREDIT_BUREAU"   // Buró de crédito
	ProviderTypeBankAPI       ProviderType = "BANK_API"        // API bancaria directa
	ProviderTypeOpenBanking   ProviderType = "OPEN_BANKING"    // Open Banking
	ProviderTypeAggregator    ProviderType = "AGGREGATOR"      // Agregador financiero
)

// ProviderConfig configuración del proveedor
type ProviderConfig struct {
	BaseURL          string            `json:"base_url"`
	Timeout          int               `json:"timeout_seconds"`
	RetryAttempts    int               `json:"retry_attempts"`
	RetryDelay       int               `json:"retry_delay_ms"`
	RateLimitPerMin  int               `json:"rate_limit_per_min"`
	CacheTTLMinutes  int               `json:"cache_ttl_minutes"`
	ResponseMapping  map[string]string `json:"response_mapping"` // Mapeo de campos del proveedor a nuestro modelo
	Headers          map[string]string `json:"headers,omitempty"`
	AuthType         string            `json:"auth_type"`        // API_KEY, OAUTH2, BASIC
}

// BankingRequest solicitud a un proveedor bancario
type BankingRequest struct {
	ID            uuid.UUID  `json:"id"`
	ApplicationID uuid.UUID  `json:"application_id"`
	ProviderID    uuid.UUID  `json:"provider_id"`
	RequestType   string     `json:"request_type"`
	Status        string     `json:"status"`        // PENDING, SUCCESS, FAILED
	RequestData   []byte     `json:"-"`             // Datos enviados (encriptados)
	ResponseData  []byte     `json:"-"`             // Respuesta (encriptada)
	ErrorMessage  string     `json:"error_message,omitempty"`
	Duration      int        `json:"duration_ms"`
	CreatedAt     time.Time  `json:"created_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

// BankingInfoResponse respuesta estandarizada de cualquier proveedor
type BankingInfoResponse struct {
	Success       bool                   `json:"success"`
	ProviderCode  string                 `json:"provider_code"`
	CreditScore   *int                   `json:"credit_score,omitempty"`
	TotalDebt     *float64               `json:"total_debt,omitempty"`
	AvailableCredit *float64             `json:"available_credit,omitempty"`
	PaymentHistory  *string              `json:"payment_history,omitempty"`
	BankAccounts  int                    `json:"bank_accounts"`
	ActiveLoans   int                    `json:"active_loans"`
	MonthsEmployed *int                  `json:"months_employed,omitempty"`
	RawData       map[string]interface{} `json:"raw_data,omitempty"`
	ErrorCode     string                 `json:"error_code,omitempty"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
}

