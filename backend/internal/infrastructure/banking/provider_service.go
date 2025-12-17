package banking

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/fintech-multipass/backend/internal/domain/entity"
	"github.com/fintech-multipass/backend/internal/infrastructure/database"
	"github.com/fintech-multipass/backend/internal/infrastructure/logger"
	"github.com/google/uuid"
)

// ProviderService servicio para integración con proveedores bancarios
type ProviderService struct {
	db  *database.PostgresDB
	log *logger.Logger
}

// NewProviderService crea una nueva instancia del servicio
func NewProviderService(db *database.PostgresDB, log *logger.Logger) *ProviderService {
	return &ProviderService{
		db:  db,
		log: log,
	}
}

// GetProviderForCountry obtiene el proveedor activo para un país
func (s *ProviderService) GetProviderForCountry(ctx context.Context, countryID uuid.UUID) (*entity.BankingProvider, error) {
	query := `
		SELECT id, country_id, code, name, type, is_active, priority, config, created_at, updated_at
		FROM banking_providers
		WHERE country_id = $1 AND is_active = true
		ORDER BY priority DESC
		LIMIT 1
	`

	var provider entity.BankingProvider
	var configJSON []byte

	row := s.db.QueryRow(ctx, query, countryID)
	err := row.Scan(
		&provider.ID, &provider.CountryID, &provider.Code, &provider.Name,
		&provider.Type, &provider.IsActive, &provider.Priority, &configJSON,
		&provider.CreatedAt, &provider.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("no active provider found for country: %w", err)
	}

	return &provider, nil
}

// FetchBankingInfo obtiene información bancaria del proveedor
// En producción, esto haría llamadas reales a APIs de burós de crédito
func (s *ProviderService) FetchBankingInfo(ctx context.Context, provider *entity.BankingProvider, docType, docNumber string) (*entity.BankingInfoResponse, error) {
	s.log.Info().
		Str("provider", provider.Code).
		Str("document", docNumber).
		Msg("Fetching banking info from provider")

	// Simular latencia de red
	time.Sleep(time.Duration(100+rand.Intn(200)) * time.Millisecond)

	// Simular respuesta del proveedor según el país
	response := s.simulateProviderResponse(provider.Code, docType, docNumber)

	s.log.Info().
		Str("provider", provider.Code).
		Bool("success", response.Success).
		Msg("Banking info fetched")

	return response, nil
}

// simulateProviderResponse simula respuestas de diferentes proveedores
func (s *ProviderService) simulateProviderResponse(providerCode, docType, docNumber string) *entity.BankingInfoResponse {
	// Generar datos simulados basados en el hash del documento
	seed := int64(0)
	for _, c := range docNumber {
		seed += int64(c)
	}
	rng := rand.New(rand.NewSource(seed))

	// Simular diferentes comportamientos por proveedor
	var response entity.BankingInfoResponse
	response.Success = true
	response.ProviderCode = providerCode

	// Generar score crediticio (300-850)
	creditScore := 300 + rng.Intn(550)
	response.CreditScore = &creditScore

	// Generar deuda total
	totalDebt := float64(rng.Intn(50000))
	response.TotalDebt = &totalDebt

	// Generar crédito disponible
	availableCredit := float64(rng.Intn(30000))
	response.AvailableCredit = &availableCredit

	// Historial de pagos
	histories := []string{"GOOD", "REGULAR", "BAD"}
	paymentHistory := histories[rng.Intn(3)]
	response.PaymentHistory = &paymentHistory

	// Cuentas y préstamos
	response.BankAccounts = rng.Intn(5)
	response.ActiveLoans = rng.Intn(3)

	// Meses empleado
	monthsEmployed := rng.Intn(120)
	response.MonthsEmployed = &monthsEmployed

	// Datos crudos simulados
	response.RawData = map[string]interface{}{
		"provider":       providerCode,
		"document_type":  docType,
		"query_date":     time.Now().Format(time.RFC3339),
		"response_code":  "OK",
		"credit_score":   creditScore,
		"debt_info":      map[string]interface{}{"total": totalDebt, "monthly_payment": totalDebt / 24},
		"employment":     map[string]interface{}{"months": monthsEmployed, "status": "EMPLOYED"},
	}

	// Simular algunos errores aleatorios (5% de probabilidad)
	if rng.Float32() < 0.05 {
		response.Success = false
		response.ErrorCode = "PROVIDER_ERROR"
		response.ErrorMessage = "Temporary service unavailability"
	}

	return &response
}

// SaveBankingInfo guarda la información bancaria obtenida
func (s *ProviderService) SaveBankingInfo(ctx context.Context, applicationID, providerID uuid.UUID, response *entity.BankingInfoResponse) error {
	info := &entity.BankingInfo{
		ID:             uuid.New(),
		ApplicationID:  applicationID,
		ProviderID:     providerID,
		CreditScore:    response.CreditScore,
		TotalDebt:      response.TotalDebt,
		AvailableCredit: response.AvailableCredit,
		PaymentHistory: response.PaymentHistory,
		BankAccounts:   response.BankAccounts,
		ActiveLoans:    response.ActiveLoans,
		MonthsEmployed: response.MonthsEmployed,
		RetrievedAt:    time.Now(),
		ExpiresAt:      time.Now().Add(24 * time.Hour), // Expira en 24 horas
	}

	query := `
		INSERT INTO banking_info (
			id, application_id, provider_id, credit_score, total_debt,
			available_credit, payment_history, bank_accounts, active_loans,
			months_employed, retrieved_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (application_id) DO UPDATE SET
			provider_id = EXCLUDED.provider_id,
			credit_score = EXCLUDED.credit_score,
			total_debt = EXCLUDED.total_debt,
			available_credit = EXCLUDED.available_credit,
			payment_history = EXCLUDED.payment_history,
			bank_accounts = EXCLUDED.bank_accounts,
			active_loans = EXCLUDED.active_loans,
			months_employed = EXCLUDED.months_employed,
			retrieved_at = EXCLUDED.retrieved_at,
			expires_at = EXCLUDED.expires_at
	`

	return s.db.Exec(ctx, query,
		info.ID, info.ApplicationID, info.ProviderID, info.CreditScore, info.TotalDebt,
		info.AvailableCredit, info.PaymentHistory, info.BankAccounts, info.ActiveLoans,
		info.MonthsEmployed, info.RetrievedAt, info.ExpiresAt,
	)
}

