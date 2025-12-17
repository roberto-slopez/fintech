package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fintech-multipass/backend/internal/domain/entity"
	"github.com/fintech-multipass/backend/internal/infrastructure/database"
	"github.com/google/uuid"
)

// ApplicationRepository implementación de repositorio de solicitudes de crédito
type ApplicationRepository struct {
	db *database.PostgresDB
}

// NewApplicationRepository crea una nueva instancia del repositorio
func NewApplicationRepository(db *database.PostgresDB) *ApplicationRepository {
	return &ApplicationRepository{db: db}
}

// Create crea una nueva solicitud de crédito
func (r *ApplicationRepository) Create(ctx context.Context, app *entity.CreditApplication) error {
	if app.ID == uuid.Nil {
		app.ID = uuid.New()
	}
	app.CreatedAt = time.Now()
	app.UpdatedAt = time.Now()
	if app.ApplicationDate.IsZero() {
		app.ApplicationDate = time.Now()
	}
	if app.Status == "" {
		app.Status = entity.StatusPending
	}

	// Ensure validation results is a valid JSON string for PostgreSQL
	var validationJSON string
	if app.ValidationResults == nil || len(app.ValidationResults) == 0 {
		validationJSON = "[]"
	} else {
		jsonBytes, err := json.Marshal(app.ValidationResults)
		if err != nil {
			validationJSON = "[]"
		} else {
			validationJSON = string(jsonBytes)
		}
	}

	query := `
		INSERT INTO credit_applications (
			id, country_id, full_name, document_type, document_number,
			email, phone, requested_amount, monthly_income, status,
			status_reason, requires_review, validation_results, risk_score,
			application_date, created_at, updated_at, created_by_ip, user_agent
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13::jsonb, $14, $15, $16, $17, $18, $19)
	`

	return r.db.Exec(ctx, query,
		app.ID, app.CountryID, app.FullName, app.DocumentType, app.DocumentNumber,
		app.Email, app.Phone, app.RequestedAmount, app.MonthlyIncome, app.Status,
		app.StatusReason, app.RequiresReview, validationJSON, app.RiskScore,
		app.ApplicationDate, app.CreatedAt, app.UpdatedAt, app.CreatedByIP, app.UserAgent,
	)
}

// GetByID obtiene una solicitud por ID
func (r *ApplicationRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.CreditApplication, error) {
	query := `
		SELECT 
			a.id, a.country_id, a.full_name, a.document_type, a.document_number,
			a.email, a.phone, a.requested_amount, a.monthly_income, a.status,
			a.status_reason, a.requires_review, a.validation_results, a.risk_score,
			a.application_date, a.processed_at, a.created_at, a.updated_at,
			c.code as country_code, c.name as country_name, c.currency
		FROM credit_applications a
		JOIN countries c ON a.country_id = c.id
		WHERE a.id = $1
	`

	var app entity.CreditApplication
	app.Country = &entity.Country{}
	var validationJSON []byte

	row := r.db.QueryRow(ctx, query, id)
	err := row.Scan(
		&app.ID, &app.CountryID, &app.FullName, &app.DocumentType, &app.DocumentNumber,
		&app.Email, &app.Phone, &app.RequestedAmount, &app.MonthlyIncome, &app.Status,
		&app.StatusReason, &app.RequiresReview, &validationJSON, &app.RiskScore,
		&app.ApplicationDate, &app.ProcessedAt, &app.CreatedAt, &app.UpdatedAt,
		&app.Country.Code, &app.Country.Name, &app.Country.Currency,
	)
	if err != nil {
		return nil, fmt.Errorf("application not found: %w", err)
	}

	if validationJSON != nil {
		_ = json.Unmarshal(validationJSON, &app.ValidationResults)
	}

	// Obtener información bancaria si existe
	bankingInfo, _ := r.GetBankingInfo(ctx, id)
	app.BankingInfo = bankingInfo

	return &app, nil
}

// Update actualiza una solicitud
func (r *ApplicationRepository) Update(ctx context.Context, app *entity.CreditApplication) error {
	app.UpdatedAt = time.Now()
	validationJSON, _ := json.Marshal(app.ValidationResults)

	query := `
		UPDATE credit_applications SET
			full_name = $2, email = $3, phone = $4,
			requested_amount = $5, monthly_income = $6, status = $7,
			status_reason = $8, requires_review = $9, validation_results = $10,
			risk_score = $11, processed_at = $12, updated_at = $13
		WHERE id = $1
	`

	return r.db.Exec(ctx, query,
		app.ID, app.FullName, app.Email, app.Phone,
		app.RequestedAmount, app.MonthlyIncome, app.Status,
		app.StatusReason, app.RequiresReview, validationJSON,
		app.RiskScore, app.ProcessedAt, app.UpdatedAt,
	)
}

// UpdateStatus actualiza solo el estado de una solicitud
func (r *ApplicationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entity.ApplicationStatus, reason string) error {
	// Convertir status a string para evitar problemas de tipo en PostgreSQL
	statusStr := string(status)
	
	query := `
		UPDATE credit_applications SET
			status = $2, status_reason = $3, updated_at = NOW(),
			processed_at = CASE WHEN $2::text IN ('APPROVED', 'REJECTED', 'DISBURSED') THEN NOW() ELSE processed_at END
		WHERE id = $1
	`
	return r.db.Exec(ctx, query, id, statusStr, reason)
}

// Delete elimina una solicitud (soft delete podría implementarse aquí)
func (r *ApplicationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM credit_applications WHERE id = $1`
	return r.db.Exec(ctx, query, id)
}

// List lista solicitudes con filtros y paginación
func (r *ApplicationRepository) List(ctx context.Context, filter entity.ApplicationFilter) (*entity.ApplicationListResult, error) {
	// Construir query dinámica
	var conditions []string
	var args []interface{}
	argIndex := 1

	baseQuery := `
		SELECT 
			a.id, a.country_id, a.full_name, a.document_type, a.document_number,
			a.email, a.phone, a.requested_amount, a.monthly_income, a.status,
			a.status_reason, a.requires_review, a.risk_score,
			a.application_date, a.processed_at, a.created_at, a.updated_at,
			c.code as country_code, c.name as country_name, c.currency
		FROM credit_applications a
		JOIN countries c ON a.country_id = c.id
	`

	// Aplicar filtros
	if filter.CountryID != nil {
		conditions = append(conditions, fmt.Sprintf("a.country_id = $%d", argIndex))
		args = append(args, *filter.CountryID)
		argIndex++
	}

	if filter.CountryCode != nil {
		conditions = append(conditions, fmt.Sprintf("c.code = $%d", argIndex))
		args = append(args, *filter.CountryCode)
		argIndex++
	}

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("a.status = $%d", argIndex))
		args = append(args, *filter.Status)
		argIndex++
	}

	if len(filter.Statuses) > 0 {
		placeholders := make([]string, len(filter.Statuses))
		for i, s := range filter.Statuses {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, s)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("a.status IN (%s)", strings.Join(placeholders, ",")))
	}

	if filter.RequiresReview != nil {
		conditions = append(conditions, fmt.Sprintf("a.requires_review = $%d", argIndex))
		args = append(args, *filter.RequiresReview)
		argIndex++
	}

	if filter.FromDate != nil {
		conditions = append(conditions, fmt.Sprintf("a.application_date >= $%d", argIndex))
		args = append(args, *filter.FromDate)
		argIndex++
	}

	if filter.ToDate != nil {
		conditions = append(conditions, fmt.Sprintf("a.application_date <= $%d", argIndex))
		args = append(args, *filter.ToDate)
		argIndex++
	}

	if filter.MinAmount != nil {
		conditions = append(conditions, fmt.Sprintf("a.requested_amount >= $%d", argIndex))
		args = append(args, *filter.MinAmount)
		argIndex++
	}

	if filter.MaxAmount != nil {
		conditions = append(conditions, fmt.Sprintf("a.requested_amount <= $%d", argIndex))
		args = append(args, *filter.MaxAmount)
		argIndex++
	}

	if filter.SearchTerm != nil && *filter.SearchTerm != "" {
		conditions = append(conditions, fmt.Sprintf("(a.full_name ILIKE $%d OR a.document_number ILIKE $%d)", argIndex, argIndex+1))
		searchPattern := "%" + *filter.SearchTerm + "%"
		args = append(args, searchPattern, searchPattern)
		argIndex += 2
	}

	// Agregar condiciones al query
	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Query para contar total
	countQuery := strings.Replace(baseQuery, 
		"SELECT \n\t\t\ta.id, a.country_id, a.full_name, a.document_type, a.document_number,\n\t\t\ta.email, a.phone, a.requested_amount, a.monthly_income, a.status,\n\t\t\ta.status_reason, a.requires_review, a.risk_score,\n\t\t\ta.application_date, a.processed_at, a.created_at, a.updated_at,\n\t\t\tc.code as country_code, c.name as country_name, c.currency",
		"SELECT COUNT(*)", 1)

	var total int64
	row := r.db.QueryRow(ctx, countQuery, args...)
	if err := row.Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count applications: %w", err)
	}

	// Ordenamiento
	sortBy := "a.created_at"
	if filter.SortBy != "" {
		allowedSorts := map[string]string{
			"created_at":       "a.created_at",
			"application_date": "a.application_date",
			"requested_amount": "a.requested_amount",
			"status":           "a.status",
		}
		if s, ok := allowedSorts[filter.SortBy]; ok {
			sortBy = s
		}
	}

	sortOrder := "DESC"
	if filter.SortOrder == "ASC" {
		sortOrder = "ASC"
	}

	baseQuery += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

	// Paginación
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	baseQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, pageSize, offset)

	// Ejecutar query
	rows, err := r.db.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query applications: %w", err)
	}
	defer rows.Close()

	var applications []entity.CreditApplication
	for rows.Next() {
		var app entity.CreditApplication
		app.Country = &entity.Country{}

		err := rows.Scan(
			&app.ID, &app.CountryID, &app.FullName, &app.DocumentType, &app.DocumentNumber,
			&app.Email, &app.Phone, &app.RequestedAmount, &app.MonthlyIncome, &app.Status,
			&app.StatusReason, &app.RequiresReview, &app.RiskScore,
			&app.ApplicationDate, &app.ProcessedAt, &app.CreatedAt, &app.UpdatedAt,
			&app.Country.Code, &app.Country.Name, &app.Country.Currency,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan application: %w", err)
		}

		applications = append(applications, app)
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &entity.ApplicationListResult{
		Applications: applications,
		Total:        total,
		Page:         page,
		PageSize:     pageSize,
		TotalPages:   totalPages,
	}, nil
}

// GetByDocumentNumber busca solicitudes por número de documento
func (r *ApplicationRepository) GetByDocumentNumber(ctx context.Context, countryID uuid.UUID, documentNumber string) ([]entity.CreditApplication, error) {
	query := `
		SELECT id, country_id, full_name, document_type, document_number,
			email, phone, requested_amount, monthly_income, status,
			status_reason, requires_review, risk_score,
			application_date, processed_at, created_at, updated_at
		FROM credit_applications
		WHERE country_id = $1 AND document_number = $2
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, countryID, documentNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to query applications: %w", err)
	}
	defer rows.Close()

	var applications []entity.CreditApplication
	for rows.Next() {
		var app entity.CreditApplication
		err := rows.Scan(
			&app.ID, &app.CountryID, &app.FullName, &app.DocumentType, &app.DocumentNumber,
			&app.Email, &app.Phone, &app.RequestedAmount, &app.MonthlyIncome, &app.Status,
			&app.StatusReason, &app.RequiresReview, &app.RiskScore,
			&app.ApplicationDate, &app.ProcessedAt, &app.CreatedAt, &app.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan application: %w", err)
		}
		applications = append(applications, app)
	}

	return applications, nil
}

// SaveStateTransition guarda una transición de estado
func (r *ApplicationRepository) SaveStateTransition(ctx context.Context, transition *entity.StateTransition) error {
	if transition.ID == uuid.Nil {
		transition.ID = uuid.New()
	}
	transition.CreatedAt = time.Now()

	query := `
		INSERT INTO state_transitions (id, application_id, from_status, to_status, reason, triggered_by, triggered_by_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	return r.db.Exec(ctx, query,
		transition.ID, transition.ApplicationID, transition.FromStatus, transition.ToStatus,
		transition.Reason, transition.TriggeredBy, transition.TriggeredByID, transition.CreatedAt,
	)
}

// GetStateTransitions obtiene el historial de transiciones de una solicitud
func (r *ApplicationRepository) GetStateTransitions(ctx context.Context, applicationID uuid.UUID) ([]entity.StateTransition, error) {
	query := `
		SELECT id, application_id, 
			COALESCE(from_status, '') as from_status, 
			to_status, 
			COALESCE(reason, '') as reason, 
			triggered_by, 
			triggered_by_id, 
			created_at
		FROM state_transitions
		WHERE application_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, applicationID)
	if err != nil {
		return nil, fmt.Errorf("failed to query transitions: %w", err)
	}
	defer rows.Close()

	var transitions []entity.StateTransition
	for rows.Next() {
		var t entity.StateTransition
		err := rows.Scan(&t.ID, &t.ApplicationID, &t.FromStatus, &t.ToStatus, &t.Reason, &t.TriggeredBy, &t.TriggeredByID, &t.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transition: %w", err)
		}
		transitions = append(transitions, t)
	}

	return transitions, nil
}

// SaveBankingInfo guarda información bancaria de una solicitud
func (r *ApplicationRepository) SaveBankingInfo(ctx context.Context, info *entity.BankingInfo) error {
	if info.ID == uuid.Nil {
		info.ID = uuid.New()
	}

	query := `
		INSERT INTO banking_info (
			id, application_id, provider_id, credit_score, total_debt,
			available_credit, payment_history, bank_accounts, active_loans,
			months_employed, raw_response, retrieved_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (application_id) DO UPDATE SET
			provider_id = EXCLUDED.provider_id,
			credit_score = EXCLUDED.credit_score,
			total_debt = EXCLUDED.total_debt,
			available_credit = EXCLUDED.available_credit,
			payment_history = EXCLUDED.payment_history,
			bank_accounts = EXCLUDED.bank_accounts,
			active_loans = EXCLUDED.active_loans,
			months_employed = EXCLUDED.months_employed,
			raw_response = EXCLUDED.raw_response,
			retrieved_at = EXCLUDED.retrieved_at,
			expires_at = EXCLUDED.expires_at
	`

	return r.db.Exec(ctx, query,
		info.ID, info.ApplicationID, info.ProviderID, info.CreditScore, info.TotalDebt,
		info.AvailableCredit, info.PaymentHistory, info.BankAccounts, info.ActiveLoans,
		info.MonthsEmployed, info.RawResponse, info.RetrievedAt, info.ExpiresAt,
	)
}

// GetBankingInfo obtiene la información bancaria de una solicitud
func (r *ApplicationRepository) GetBankingInfo(ctx context.Context, applicationID uuid.UUID) (*entity.BankingInfo, error) {
	query := `
		SELECT bi.id, bi.application_id, bi.provider_id, bi.credit_score, bi.total_debt,
			bi.available_credit, bi.payment_history, bi.bank_accounts, bi.active_loans,
			bi.months_employed, bi.retrieved_at, bi.expires_at, bp.name as provider_name
		FROM banking_info bi
		JOIN banking_providers bp ON bi.provider_id = bp.id
		WHERE bi.application_id = $1
	`

	var info entity.BankingInfo
	row := r.db.QueryRow(ctx, query, applicationID)
	err := row.Scan(
		&info.ID, &info.ApplicationID, &info.ProviderID, &info.CreditScore, &info.TotalDebt,
		&info.AvailableCredit, &info.PaymentHistory, &info.BankAccounts, &info.ActiveLoans,
		&info.MonthsEmployed, &info.RetrievedAt, &info.ExpiresAt, &info.ProviderName,
	)
	if err != nil {
		return nil, err
	}

	return &info, nil
}

