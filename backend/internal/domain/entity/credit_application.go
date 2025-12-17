package entity

import (
	"time"

	"github.com/google/uuid"
)

// CreditApplication representa una solicitud de crédito
type CreditApplication struct {
	ID              uuid.UUID          `json:"id"`
	CountryID       uuid.UUID          `json:"country_id"`
	Country         *Country           `json:"country,omitempty"`
	
	// Datos del solicitante
	FullName        string             `json:"full_name"`
	DocumentType    string             `json:"document_type"`    // DNI, NIF, CURP, etc.
	DocumentNumber  string             `json:"document_number"`
	Email           string             `json:"email"`
	Phone           string             `json:"phone,omitempty"`
	
	// Datos financieros
	RequestedAmount float64            `json:"requested_amount"`
	MonthlyIncome   float64            `json:"monthly_income"`
	
	// Estado y flujo
	Status          ApplicationStatus  `json:"status"`
	StatusReason    string             `json:"status_reason,omitempty"`
	RequiresReview  bool               `json:"requires_review"`
	
	// Información bancaria del proveedor
	BankingInfo     *BankingInfo       `json:"banking_info,omitempty"`
	
	// Resultados de validación
	ValidationResults []ValidationResult `json:"validation_results,omitempty"`
	RiskScore       *float64           `json:"risk_score,omitempty"`
	
	// Metadatos
	ApplicationDate time.Time          `json:"application_date"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
	ProcessedAt     *time.Time         `json:"processed_at,omitempty"`
	
	// Para auditoría (no expuestos en API pública)
	CreatedByIP     string             `json:"-"`
	UserAgent       string             `json:"-"`
}

// ApplicationStatus estados posibles de una solicitud
type ApplicationStatus string

const (
	StatusPending          ApplicationStatus = "PENDING"           // Recién creada
	StatusValidating       ApplicationStatus = "VALIDATING"        // En proceso de validación
	StatusPendingBankInfo  ApplicationStatus = "PENDING_BANK_INFO" // Esperando info bancaria
	StatusUnderReview      ApplicationStatus = "UNDER_REVIEW"      // Requiere revisión manual
	StatusApproved         ApplicationStatus = "APPROVED"          // Aprobada
	StatusRejected         ApplicationStatus = "REJECTED"          // Rechazada
	StatusCancelled        ApplicationStatus = "CANCELLED"         // Cancelada
	StatusExpired          ApplicationStatus = "EXPIRED"           // Expirada
	StatusDisbursed        ApplicationStatus = "DISBURSED"         // Desembolsada
)

// IsTerminal verifica si el estado es terminal (no puede cambiar más)
func (s ApplicationStatus) IsTerminal() bool {
	return s == StatusApproved || s == StatusRejected || 
	       s == StatusCancelled || s == StatusExpired || s == StatusDisbursed
}

// CanTransitionTo verifica si se puede transicionar a otro estado
func (s ApplicationStatus) CanTransitionTo(target ApplicationStatus) bool {
	transitions := map[ApplicationStatus][]ApplicationStatus{
		StatusPending:         {StatusValidating, StatusCancelled},
		StatusValidating:      {StatusPendingBankInfo, StatusUnderReview, StatusApproved, StatusRejected},
		StatusPendingBankInfo: {StatusValidating, StatusUnderReview, StatusRejected, StatusCancelled},
		StatusUnderReview:     {StatusApproved, StatusRejected, StatusCancelled},
		StatusApproved:        {StatusDisbursed, StatusCancelled, StatusExpired},
		StatusRejected:        {}, // Terminal
		StatusCancelled:       {}, // Terminal
		StatusExpired:         {}, // Terminal
		StatusDisbursed:       {}, // Terminal
	}
	
	allowed, exists := transitions[s]
	if !exists {
		return false
	}
	
	for _, t := range allowed {
		if t == target {
			return true
		}
	}
	return false
}

// StateTransition representa un cambio de estado en una solicitud
type StateTransition struct {
	ID              uuid.UUID         `json:"id"`
	ApplicationID   uuid.UUID         `json:"application_id"`
	FromStatus      ApplicationStatus `json:"from_status"`
	ToStatus        ApplicationStatus `json:"to_status"`
	Reason          string            `json:"reason,omitempty"`
	TriggeredBy     string            `json:"triggered_by"` // SYSTEM, USER, WEBHOOK
	TriggeredByID   *uuid.UUID        `json:"triggered_by_id,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
}

// BankingInfo información bancaria obtenida del proveedor
type BankingInfo struct {
	ID                uuid.UUID  `json:"id"`
	ApplicationID     uuid.UUID  `json:"application_id"`
	ProviderID        uuid.UUID  `json:"provider_id"`
	ProviderName      string     `json:"provider_name"`
	
	// Datos obtenidos del proveedor (varían por país/proveedor)
	CreditScore       *int       `json:"credit_score,omitempty"`
	TotalDebt         *float64   `json:"total_debt,omitempty"`
	AvailableCredit   *float64   `json:"available_credit,omitempty"`
	PaymentHistory    *string    `json:"payment_history,omitempty"` // GOOD, REGULAR, BAD
	BankAccounts      int        `json:"bank_accounts"`
	ActiveLoans       int        `json:"active_loans"`
	MonthsEmployed    *int       `json:"months_employed,omitempty"`
	
	// Datos crudos del proveedor (para auditoría, no expuestos)
	RawResponse       []byte     `json:"-"`
	
	RetrievedAt       time.Time  `json:"retrieved_at"`
	ExpiresAt         time.Time  `json:"expires_at"`
}

// ApplicationFilter filtros para búsqueda de solicitudes
type ApplicationFilter struct {
	CountryID     *uuid.UUID
	CountryCode   *string
	Status        *ApplicationStatus
	Statuses      []ApplicationStatus
	RequiresReview *bool
	FromDate      *time.Time
	ToDate        *time.Time
	MinAmount     *float64
	MaxAmount     *float64
	SearchTerm    *string // Búsqueda en nombre o documento
	
	// Paginación
	Page          int
	PageSize      int
	SortBy        string
	SortOrder     string // ASC, DESC
}

// ListResult resultado paginado de solicitudes
type ApplicationListResult struct {
	Applications []CreditApplication `json:"applications"`
	Total        int64               `json:"total"`
	Page         int                 `json:"page"`
	PageSize     int                 `json:"page_size"`
	TotalPages   int                 `json:"total_pages"`
}

// Nota: ValidationResult, BankingInfoResponse, RuleType y CountryRule 
// están definidos en country.go y banking_provider.go

