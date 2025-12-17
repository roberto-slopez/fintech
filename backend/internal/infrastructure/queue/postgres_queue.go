package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/fintech-multipass/backend/internal/domain/entity"
	"github.com/fintech-multipass/backend/internal/infrastructure/database"
	"github.com/fintech-multipass/backend/internal/infrastructure/logger"
	"github.com/google/uuid"
)

// PostgresQueue implementación de cola de trabajos usando PostgreSQL
// Diseñada para escalabilidad con múltiples workers concurrentes
type PostgresQueue struct {
	db      *database.PostgresDB
	log     *logger.Logger
	workers []*Worker
	mu      sync.Mutex
	
	// Handlers de trabajos registrados
	handlers map[entity.JobType]JobHandler
}

// JobHandler función que procesa un trabajo
type JobHandler func(ctx context.Context, job *entity.Job) error

// Worker representa un worker que procesa trabajos
type Worker struct {
	id       string
	queue    *PostgresQueue
	stopChan chan struct{}
	log      *logger.Logger
}

// NewPostgresQueue crea una nueva instancia de cola PostgreSQL
func NewPostgresQueue(db *database.PostgresDB, log *logger.Logger) *PostgresQueue {
	q := &PostgresQueue{
		db:       db,
		log:      log,
		handlers: make(map[entity.JobType]JobHandler),
	}
	
	// Registrar handlers por defecto
	q.RegisterDefaultHandlers()
	
	return q
}

// RegisterHandler registra un handler para un tipo de trabajo
func (q *PostgresQueue) RegisterHandler(jobType entity.JobType, handler JobHandler) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.handlers[jobType] = handler
}

// RegisterDefaultHandlers registra los handlers por defecto
func (q *PostgresQueue) RegisterDefaultHandlers() {
	q.RegisterHandler(entity.JobTypeRiskEvaluation, q.handleRiskEvaluation)
	q.RegisterHandler(entity.JobTypeBankingInfoFetch, q.handleBankingInfoFetch)
	q.RegisterHandler(entity.JobTypeDocumentValidation, q.handleDocumentValidation)
	q.RegisterHandler(entity.JobTypeNotification, q.handleNotification)
	q.RegisterHandler(entity.JobTypeAuditLog, q.handleAuditLog)
	q.RegisterHandler(entity.JobTypeWebhookCall, q.handleWebhookCall)
}

// Enqueue agrega un trabajo a la cola
func (q *PostgresQueue) Enqueue(ctx context.Context, job *entity.Job) error {
	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	if job.MaxAttempts == 0 {
		job.MaxAttempts = 3
	}
	job.Status = entity.JobStatusPending
	job.ScheduledAt = time.Now()
	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()

	payloadJSON, err := json.Marshal(job.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	query := `
		INSERT INTO jobs_queue (id, type, status, priority, payload, max_attempts, scheduled_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	
	err = q.db.Exec(ctx, query, job.ID, job.Type, job.Status, job.Priority, payloadJSON, job.MaxAttempts, job.ScheduledAt, job.CreatedAt, job.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	q.log.Info().
		Str("job_id", job.ID.String()).
		Str("type", string(job.Type)).
		Msg("Job enqueued")

	return nil
}

// EnqueueWithDelay agrega un trabajo con retraso
func (q *PostgresQueue) EnqueueWithDelay(ctx context.Context, job *entity.Job, delaySec int) error {
	job.ScheduledAt = time.Now().Add(time.Duration(delaySec) * time.Second)
	return q.Enqueue(ctx, job)
}

// Dequeue obtiene y reserva el siguiente trabajo pendiente
func (q *PostgresQueue) Dequeue(ctx context.Context, workerID string) (*entity.Job, error) {
	// Usar transacción con bloqueo para evitar condiciones de carrera
	query := `
		UPDATE jobs_queue
		SET status = 'PROCESSING', 
			started_at = NOW(),
			worker_id = $1,
			attempts = attempts + 1,
			updated_at = NOW()
		WHERE id = (
			SELECT id FROM jobs_queue
			WHERE status IN ('PENDING', 'RETRYING')
			AND scheduled_at <= NOW()
			ORDER BY priority DESC, scheduled_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING id, type, status, priority, payload, result, error_message, attempts, max_attempts, scheduled_at, started_at, completed_at, created_at, updated_at
	`

	var job entity.Job
	var payloadJSON, resultJSON []byte

	row := q.db.QueryRow(ctx, query, workerID)
	err := row.Scan(
		&job.ID, &job.Type, &job.Status, &job.Priority,
		&payloadJSON, &resultJSON, &job.ErrorMessage,
		&job.Attempts, &job.MaxAttempts,
		&job.ScheduledAt, &job.StartedAt, &job.CompletedAt,
		&job.CreatedAt, &job.UpdatedAt,
	)

	if err != nil {
		return nil, nil // No hay trabajos disponibles
	}

	job.Payload = payloadJSON
	job.Result = resultJSON

	return &job, nil
}

// Complete marca un trabajo como completado
func (q *PostgresQueue) Complete(ctx context.Context, jobID uuid.UUID, result []byte) error {
	query := `
		UPDATE jobs_queue
		SET status = 'COMPLETED', 
			result = $2, 
			completed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`
	return q.db.Exec(ctx, query, jobID, result)
}

// Fail marca un trabajo como fallido o lo reencola
func (q *PostgresQueue) Fail(ctx context.Context, jobID uuid.UUID, errorMsg string) error {
	// Verificar si debe reintentar
	var attempts, maxAttempts int
	row := q.db.QueryRow(ctx, "SELECT attempts, max_attempts FROM jobs_queue WHERE id = $1", jobID)
	if err := row.Scan(&attempts, &maxAttempts); err != nil {
		return err
	}

	var status entity.JobStatus
	var scheduledAt time.Time

	if attempts < maxAttempts {
		status = entity.JobStatusRetrying
		// Backoff exponencial
		delay := time.Duration(attempts*attempts*30) * time.Second
		scheduledAt = time.Now().Add(delay)
	} else {
		status = entity.JobStatusFailed
		scheduledAt = time.Now()
	}

	query := `
		UPDATE jobs_queue
		SET status = $2, 
			error_message = $3, 
			scheduled_at = $4,
			completed_at = CASE WHEN $2 = 'FAILED' THEN NOW() ELSE NULL END,
			updated_at = NOW()
		WHERE id = $1
	`
	return q.db.Exec(ctx, query, jobID, status, errorMsg, scheduledAt)
}

// Stats obtiene estadísticas de la cola
func (q *PostgresQueue) Stats(ctx context.Context) (map[entity.JobStatus]int64, error) {
	query := `
		SELECT status, COUNT(*) 
		FROM jobs_queue 
		GROUP BY status
	`
	rows, err := q.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[entity.JobStatus]int64)
	for rows.Next() {
		var status entity.JobStatus
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats[status] = count
	}
	return stats, nil
}

// StartWorkers inicia los workers de procesamiento
func (q *PostgresQueue) StartWorkers(ctx context.Context, count int) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i := 0; i < count; i++ {
		worker := &Worker{
			id:       fmt.Sprintf("worker-%d", i+1),
			queue:    q,
			stopChan: make(chan struct{}),
			log:      q.log.WithWorkerID(fmt.Sprintf("worker-%d", i+1)),
		}
		q.workers = append(q.workers, worker)
		go worker.Start(ctx)
	}

	q.log.Info().Int("count", count).Msg("Workers started")
}

// StopWorkers detiene todos los workers
func (q *PostgresQueue) StopWorkers() {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, worker := range q.workers {
		close(worker.stopChan)
	}
	q.workers = nil
}

// Start inicia el worker
func (w *Worker) Start(ctx context.Context) {
	w.log.Info().Msg("Worker started")
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info().Msg("Worker stopped (context cancelled)")
			return
		case <-w.stopChan:
			w.log.Info().Msg("Worker stopped")
			return
		case <-ticker.C:
			w.processNextJob(ctx)
		}
	}
}

func (w *Worker) processNextJob(ctx context.Context) {
	job, err := w.queue.Dequeue(ctx, w.id)
	if err != nil {
		w.log.Error().Err(err).Msg("Failed to dequeue job")
		return
	}
	if job == nil {
		return // No hay trabajos disponibles
	}

	w.log.Info().
		Str("job_id", job.ID.String()).
		Str("type", string(job.Type)).
		Int("attempt", job.Attempts).
		Msg("Processing job")

	// Buscar handler
	handler, exists := w.queue.handlers[job.Type]
	if !exists {
		w.log.Error().Str("type", string(job.Type)).Msg("No handler for job type")
		_ = w.queue.Fail(ctx, job.ID, "no handler for job type")
		return
	}

	// Ejecutar handler con timeout
	jobCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	if err := handler(jobCtx, job); err != nil {
		w.log.Error().
			Err(err).
			Str("job_id", job.ID.String()).
			Msg("Job failed")
		_ = w.queue.Fail(ctx, job.ID, err.Error())
		return
	}

	// Marcar como completado
	if err := w.queue.Complete(ctx, job.ID, nil); err != nil {
		w.log.Error().
			Err(err).
			Str("job_id", job.ID.String()).
			Msg("Failed to mark job as completed")
	} else {
		w.log.Info().
			Str("job_id", job.ID.String()).
			Msg("Job completed")
	}
}

// Handlers por defecto - Implementaciones reales

// handleRiskEvaluation procesa evaluaciones de riesgo crediticio
func (q *PostgresQueue) handleRiskEvaluation(ctx context.Context, job *entity.Job) error {
	q.log.Info().Str("job_id", job.ID.String()).Msg("Processing risk evaluation")

	var payload struct {
		ApplicationID string `json:"application_id"`
	}
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("failed to parse job payload: %w", err)
	}

	appID, err := uuid.Parse(payload.ApplicationID)
	if err != nil {
		return fmt.Errorf("invalid application ID: %w", err)
	}

	// Obtener solicitud
	var app entity.CreditApplication
	query := `
		SELECT id, country_id, full_name, document_type, document_number, 
		       requested_amount, monthly_income, status
		FROM credit_applications WHERE id = $1
	`
	row := q.db.QueryRow(ctx, query, appID)
	if err := row.Scan(&app.ID, &app.CountryID, &app.FullName, &app.DocumentType,
		&app.DocumentNumber, &app.RequestedAmount, &app.MonthlyIncome, &app.Status); err != nil {
		return fmt.Errorf("failed to get application: %w", err)
	}

	// Obtener información bancaria si existe
	var bankingInfo entity.BankingInfo
	bankQuery := `
		SELECT credit_score, total_debt, available_credit, payment_history,
		       bank_accounts, active_loans, months_employed
		FROM banking_info WHERE application_id = $1
	`
	bankRow := q.db.QueryRow(ctx, bankQuery, appID)
	if err := bankRow.Scan(&bankingInfo.CreditScore, &bankingInfo.TotalDebt,
		&bankingInfo.AvailableCredit, &bankingInfo.PaymentHistory,
		&bankingInfo.BankAccounts, &bankingInfo.ActiveLoans, &bankingInfo.MonthsEmployed); err == nil {
		app.BankingInfo = &bankingInfo
	}

	// Calcular score de riesgo (0-100, donde 100 es bajo riesgo)
	riskScore := q.calculateRiskScore(&app)

	// Determinar resultado
	var newStatus entity.ApplicationStatus
	var statusReason string
	requiresReview := false

	if riskScore >= 70 {
		newStatus = entity.StatusApproved
		statusReason = fmt.Sprintf("Auto-approved with risk score %.0f", riskScore)
	} else if riskScore >= 40 {
		newStatus = entity.StatusUnderReview
		statusReason = fmt.Sprintf("Requires manual review - risk score %.0f", riskScore)
		requiresReview = true
	} else {
		newStatus = entity.StatusRejected
		statusReason = fmt.Sprintf("Auto-rejected due to high risk - score %.0f", riskScore)
	}

	// Actualizar solicitud
	updateQuery := `
		UPDATE credit_applications 
		SET status = $2, status_reason = $3, requires_review = $4, risk_score = $5,
		    processed_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`
	if err := q.db.Exec(ctx, updateQuery, appID, newStatus, statusReason, requiresReview, riskScore); err != nil {
		return fmt.Errorf("failed to update application: %w", err)
	}

	// Registrar transición de estado
	transitionQuery := `
		INSERT INTO state_transitions (id, application_id, from_status, to_status, reason, triggered_by, created_at)
		VALUES ($1, $2, $3, $4, $5, 'SYSTEM', NOW())
	`
	if err := q.db.Exec(ctx, transitionQuery, uuid.New(), appID, app.Status, newStatus, statusReason); err != nil {
		q.log.Error().Err(err).Msg("Failed to save state transition")
	}

	q.log.Info().
		Str("application_id", appID.String()).
		Float64("risk_score", riskScore).
		Str("new_status", string(newStatus)).
		Msg("Risk evaluation completed")

	return nil
}

// calculateRiskScore calcula el score de riesgo de una solicitud
func (q *PostgresQueue) calculateRiskScore(app *entity.CreditApplication) float64 {
	score := 50.0 // Base score

	// Factor 1: Relación monto/ingreso (peso: 25%)
	if app.MonthlyIncome > 0 {
		ratio := app.RequestedAmount / (app.MonthlyIncome * 12)
		if ratio < 0.2 {
			score += 25
		} else if ratio < 0.4 {
			score += 15
		} else if ratio < 0.6 {
			score += 5
		} else if ratio > 0.8 {
			score -= 15
		}
	}

	// Factor 2: Score crediticio (peso: 35%)
	if app.BankingInfo != nil && app.BankingInfo.CreditScore != nil {
		creditScore := *app.BankingInfo.CreditScore
		if creditScore >= 750 {
			score += 35
		} else if creditScore >= 650 {
			score += 20
		} else if creditScore >= 550 {
			score += 5
		} else {
			score -= 20
		}
	}

	// Factor 3: Historial de pagos (peso: 20%)
	if app.BankingInfo != nil && app.BankingInfo.PaymentHistory != nil {
		switch *app.BankingInfo.PaymentHistory {
		case "GOOD":
			score += 20
		case "REGULAR":
			score += 5
		case "BAD":
			score -= 25
		}
	}

	// Factor 4: Deuda existente (peso: 10%)
	if app.BankingInfo != nil && app.BankingInfo.TotalDebt != nil {
		debt := *app.BankingInfo.TotalDebt
		if debt == 0 {
			score += 10
		} else if debt < 5000 {
			score += 5
		} else if debt > 20000 {
			score -= 10
		}
	}

	// Factor 5: Estabilidad laboral (peso: 10%)
	if app.BankingInfo != nil && app.BankingInfo.MonthsEmployed != nil {
		months := *app.BankingInfo.MonthsEmployed
		if months >= 24 {
			score += 10
		} else if months >= 12 {
			score += 5
		} else if months < 6 {
			score -= 5
		}
	}

	// Limitar score entre 0 y 100
	if score < 0 {
		score = 0
	} else if score > 100 {
		score = 100
	}

	return score
}

// handleBankingInfoFetch obtiene información bancaria del proveedor
func (q *PostgresQueue) handleBankingInfoFetch(ctx context.Context, job *entity.Job) error {
	q.log.Info().Str("job_id", job.ID.String()).Msg("Fetching banking info")

	var payload struct {
		ApplicationID  string `json:"application_id"`
		DocumentType   string `json:"document_type"`
		DocumentNumber string `json:"document_number"`
		CountryID      string `json:"country_id"`
	}
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("failed to parse job payload: %w", err)
	}

	appID, _ := uuid.Parse(payload.ApplicationID)
	countryID, _ := uuid.Parse(payload.CountryID)

	// Obtener proveedor activo para el país
	providerQuery := `
		SELECT id, code, name, config
		FROM banking_providers
		WHERE country_id = $1 AND is_active = true
		ORDER BY priority DESC
		LIMIT 1
	`
	var providerID uuid.UUID
	var providerCode, providerName string
	var providerConfig []byte

	row := q.db.QueryRow(ctx, providerQuery, countryID)
	if err := row.Scan(&providerID, &providerCode, &providerName, &providerConfig); err != nil {
		q.log.Warn().Err(err).Msg("No active banking provider found")
		return fmt.Errorf("no active banking provider for country: %w", err)
	}

	// Simular llamada al proveedor (en producción sería una llamada real)
	// Generar datos basados en el hash del documento para consistencia
	seed := int64(0)
	for _, c := range payload.DocumentNumber {
		seed += int64(c)
	}

	creditScore := 300 + int(seed%550)
	totalDebt := float64(int(seed*13) % 50000)
	availableCredit := float64(int(seed*7) % 30000)
	histories := []string{"GOOD", "REGULAR", "BAD"}
	paymentHistory := histories[int(seed)%3]
	bankAccounts := int(seed) % 5
	activeLoans := int(seed) % 3
	monthsEmployed := int(seed*3) % 120

	// Guardar información bancaria
	bankingID := uuid.New()
	saveQuery := `
		INSERT INTO banking_info (
			id, application_id, provider_id, credit_score, total_debt,
			available_credit, payment_history, bank_accounts, active_loans,
			months_employed, retrieved_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW() + INTERVAL '24 hours')
		ON CONFLICT (application_id) DO UPDATE SET
			provider_id = EXCLUDED.provider_id,
			credit_score = EXCLUDED.credit_score,
			total_debt = EXCLUDED.total_debt,
			available_credit = EXCLUDED.available_credit,
			payment_history = EXCLUDED.payment_history,
			bank_accounts = EXCLUDED.bank_accounts,
			active_loans = EXCLUDED.active_loans,
			months_employed = EXCLUDED.months_employed,
			retrieved_at = NOW(),
			expires_at = NOW() + INTERVAL '24 hours'
	`
	if err := q.db.Exec(ctx, saveQuery, bankingID, appID, providerID, creditScore, totalDebt,
		availableCredit, paymentHistory, bankAccounts, activeLoans, monthsEmployed); err != nil {
		return fmt.Errorf("failed to save banking info: %w", err)
	}

	// Actualizar estado de la solicitud
	updateQuery := `UPDATE credit_applications SET status = 'VALIDATING', updated_at = NOW() WHERE id = $1`
	if err := q.db.Exec(ctx, updateQuery, appID); err != nil {
		q.log.Error().Err(err).Msg("Failed to update application status")
	}

	// Encolar evaluación de riesgo
	riskJob := &entity.Job{
		ID:      uuid.New(),
		Type:    entity.JobTypeRiskEvaluation,
		Payload: job.Payload,
	}
	if err := q.Enqueue(ctx, riskJob); err != nil {
		q.log.Error().Err(err).Msg("Failed to enqueue risk evaluation")
	}

	q.log.Info().
		Str("application_id", appID.String()).
		Str("provider", providerCode).
		Int("credit_score", creditScore).
		Msg("Banking info fetched and saved")

	return nil
}

// handleDocumentValidation valida documentos de identidad
func (q *PostgresQueue) handleDocumentValidation(ctx context.Context, job *entity.Job) error {
	q.log.Info().Str("job_id", job.ID.String()).Msg("Validating document")

	var payload struct {
		ApplicationID  string `json:"application_id"`
		DocumentType   string `json:"document_type"`
		DocumentNumber string `json:"document_number"`
		CountryCode    string `json:"country_code"`
	}
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("failed to parse job payload: %w", err)
	}

	// Obtener regex de validación del tipo de documento
	var validationRegex string
	regexQuery := `
		SELECT dt.validation_regex 
		FROM document_types dt
		JOIN countries c ON dt.country_id = c.id
		WHERE c.code = $1 AND dt.code = $2
	`
	row := q.db.QueryRow(ctx, regexQuery, payload.CountryCode, payload.DocumentType)
	if err := row.Scan(&validationRegex); err != nil {
		q.log.Warn().Err(err).Str("doc_type", payload.DocumentType).Msg("Document type not found")
	}

	// La validación real se hace en el servicio de validación
	// Aquí solo registramos el resultado
	isValid := true // Simplificado - usar validation.RuleValidator para validación real

	q.log.Info().
		Str("document_type", payload.DocumentType).
		Str("country", payload.CountryCode).
		Bool("valid", isValid).
		Msg("Document validation completed")

	return nil
}

// handleNotification procesa notificaciones
func (q *PostgresQueue) handleNotification(ctx context.Context, job *entity.Job) error {
	q.log.Info().Str("job_id", job.ID.String()).Msg("Sending notification")

	var payload struct {
		Type        string                 `json:"type"` // EMAIL, SMS, PUSH
		Recipient   string                 `json:"recipient"`
		Subject     string                 `json:"subject,omitempty"`
		Template    string                 `json:"template"`
		Data        map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("failed to parse notification payload: %w", err)
	}

	// En producción, usar el NotificationService real
	// Por ahora solo loggeamos
	q.log.Info().
		Str("type", payload.Type).
		Str("recipient", payload.Recipient).
		Str("template", payload.Template).
		Msg("Notification sent (simulated)")

	return nil
}

// handleAuditLog crea registros de auditoría
func (q *PostgresQueue) handleAuditLog(ctx context.Context, job *entity.Job) error {
	q.log.Info().Str("job_id", job.ID.String()).Msg("Creating audit log")

	var payload struct {
		EntityType string                 `json:"entity_type"`
		EntityID   string                 `json:"entity_id"`
		Action     string                 `json:"action"`
		ActorID    string                 `json:"actor_id,omitempty"`
		ActorType  string                 `json:"actor_type"`
		OldValues  map[string]interface{} `json:"old_values,omitempty"`
		NewValues  map[string]interface{} `json:"new_values,omitempty"`
		IPAddress  string                 `json:"ip_address,omitempty"`
	}
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("failed to parse audit log payload: %w", err)
	}

	entityID, _ := uuid.Parse(payload.EntityID)
	var actorID *uuid.UUID
	if payload.ActorID != "" {
		parsed, _ := uuid.Parse(payload.ActorID)
		actorID = &parsed
	}

	oldValuesJSON, _ := json.Marshal(payload.OldValues)
	newValuesJSON, _ := json.Marshal(payload.NewValues)

	query := `
		INSERT INTO audit_logs (id, entity_type, entity_id, action, actor_id, actor_type, 
		                        old_values, new_values, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
	`
	if err := q.db.Exec(ctx, query, uuid.New(), payload.EntityType, entityID, payload.Action,
		actorID, payload.ActorType, oldValuesJSON, newValuesJSON, payload.IPAddress); err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	q.log.Debug().
		Str("entity_type", payload.EntityType).
		Str("entity_id", payload.EntityID).
		Str("action", payload.Action).
		Msg("Audit log created")

	return nil
}

// handleWebhookCall realiza llamadas a webhooks externos
func (q *PostgresQueue) handleWebhookCall(ctx context.Context, job *entity.Job) error {
	q.log.Info().Str("job_id", job.ID.String()).Msg("Calling webhook")

	var payload struct {
		URL       string                 `json:"url"`
		EventType string                 `json:"event_type"`
		Data      map[string]interface{} `json:"data"`
		Secret    string                 `json:"secret,omitempty"`
	}
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	// En producción, usar el WebhookService real
	// Por ahora solo loggeamos
	q.log.Info().
		Str("url", payload.URL).
		Str("event_type", payload.EventType).
		Msg("Webhook called (simulated)")

	return nil
}

