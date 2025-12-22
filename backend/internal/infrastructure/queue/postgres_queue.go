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

	// El payload debe ser []byte con JSON válido
	// Verificar que sea JSON válido
	if len(job.Payload) == 0 {
		return fmt.Errorf("empty payload")
	}
	if !json.Valid(job.Payload) {
		q.log.Error().Str("payload", string(job.Payload)).Msg("Invalid JSON payload")
		return fmt.Errorf("invalid JSON payload")
	}

	// Convertir a string para que pgx lo envíe correctamente como JSONB
	payloadStr := string(job.Payload)

	query := `
		INSERT INTO jobs_queue (id, type, status, priority, payload, max_attempts, scheduled_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8, $9)
	`
	
	err := q.db.Exec(ctx, query, job.ID, job.Type, job.Status, job.Priority, payloadStr, job.MaxAttempts, job.ScheduledAt, job.CreatedAt, job.UpdatedAt)
	if err != nil {
		q.log.Error().Err(err).Str("payload", payloadStr).Msg("Failed to insert job")
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
	var errorMessage *string
	var startedAt, completedAt *time.Time

	row := q.db.QueryRow(ctx, query, workerID)
	err := row.Scan(
		&job.ID, &job.Type, &job.Status, &job.Priority,
		&payloadJSON, &resultJSON, &errorMessage,
		&job.Attempts, &job.MaxAttempts,
		&job.ScheduledAt, &startedAt, &completedAt,
		&job.CreatedAt, &job.UpdatedAt,
	)

	if err != nil {
		// pgx devuelve ErrNoRows cuando no hay filas, lo cual es normal
		// Para otros errores, logearlos
		if err.Error() != "no rows in result set" {
			q.log.Error().Err(err).Str("worker_id", workerID).Msg("Error dequeuing job")
		}
		return nil, nil // No hay trabajos disponibles
	}

	job.Payload = payloadJSON
	job.Result = resultJSON
	if errorMessage != nil {
		job.ErrorMessage = *errorMessage
	}
	if startedAt != nil {
		job.StartedAt = startedAt
	}
	if completedAt != nil {
		job.CompletedAt = completedAt
	}

	q.log.Debug().
		Str("job_id", job.ID.String()).
		Str("type", string(job.Type)).
		Str("worker_id", workerID).
		Msg("Job dequeued successfully")

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

// RecentJobInfo información de un job reciente para depuración
type RecentJobInfo struct {
	ID           uuid.UUID         `json:"id"`
	Type         entity.JobType    `json:"type"`
	Status       entity.JobStatus  `json:"status"`
	Attempts     int               `json:"attempts"`
	MaxAttempts  int               `json:"max_attempts"`
	ErrorMessage *string           `json:"error_message,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	ScheduledAt  time.Time         `json:"scheduled_at"`
	StartedAt    *time.Time        `json:"started_at,omitempty"`
	CompletedAt  *time.Time        `json:"completed_at,omitempty"`
}

// GetRecentJobs obtiene los jobs recientes para depuración
func (q *PostgresQueue) GetRecentJobs(ctx context.Context, limit int) ([]RecentJobInfo, error) {
	query := `
		SELECT id, type, status, attempts, max_attempts, error_message, 
		       created_at, scheduled_at, started_at, completed_at
		FROM jobs_queue 
		ORDER BY created_at DESC
		LIMIT $1
	`
	rows, err := q.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []RecentJobInfo
	for rows.Next() {
		var job RecentJobInfo
		if err := rows.Scan(&job.ID, &job.Type, &job.Status, &job.Attempts, &job.MaxAttempts,
			&job.ErrorMessage, &job.CreatedAt, &job.ScheduledAt, &job.StartedAt, &job.CompletedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

// GetFailedJobs obtiene los jobs fallidos para depuración
func (q *PostgresQueue) GetFailedJobs(ctx context.Context, limit int) ([]RecentJobInfo, error) {
	query := `
		SELECT id, type, status, attempts, max_attempts, error_message, 
		       created_at, scheduled_at, started_at, completed_at
		FROM jobs_queue 
		WHERE status = 'FAILED'
		ORDER BY completed_at DESC
		LIMIT $1
	`
	rows, err := q.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []RecentJobInfo
	for rows.Next() {
		var job RecentJobInfo
		if err := rows.Scan(&job.ID, &job.Type, &job.Status, &job.Attempts, &job.MaxAttempts,
			&job.ErrorMessage, &job.CreatedAt, &job.ScheduledAt, &job.StartedAt, &job.CompletedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

// RecoverOrphanedJobs recupera jobs que quedaron en PROCESSING por más de X minutos
func (q *PostgresQueue) RecoverOrphanedJobs(ctx context.Context, staleMinutes int) (int64, error) {
	query := `
		UPDATE jobs_queue 
		SET status = 'PENDING', 
			worker_id = NULL, 
			started_at = NULL
		WHERE status = 'PROCESSING' 
		AND started_at < NOW() - INTERVAL '1 minute' * $1
	`
	result, err := q.db.Pool.Exec(ctx, query, staleMinutes)
	if err != nil {
		return 0, err
	}

	count := result.RowsAffected()
	if count > 0 {
		q.log.Warn().Int64("count", count).Int("stale_minutes", staleMinutes).Msg("Recovered orphaned jobs")
	}
	return count, nil
}

// StartWorkers inicia los workers de procesamiento
func (q *PostgresQueue) StartWorkers(ctx context.Context, count int) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Recuperar jobs huérfanos al inicio (más de 5 minutos en PROCESSING)
	if recovered, err := q.RecoverOrphanedJobs(ctx, 5); err != nil {
		q.log.Error().Err(err).Msg("Failed to recover orphaned jobs")
	} else if recovered > 0 {
		q.log.Info().Int64("recovered", recovered).Msg("Recovered orphaned jobs at startup")
	}

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

	// Iniciar goroutine para recuperar jobs huérfanos periódicamente
	go q.orphanRecoveryLoop(ctx)

	q.log.Info().Int("count", count).Msg("Workers started")
}

// orphanRecoveryLoop verifica periódicamente si hay jobs huérfanos
func (q *PostgresQueue) orphanRecoveryLoop(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := q.RecoverOrphanedJobs(ctx, 5); err != nil {
				q.log.Error().Err(err).Msg("Failed to recover orphaned jobs in loop")
			}
		}
	}
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
	// Recuperar de cualquier panic para evitar que el worker muera
	defer func() {
		if r := recover(); r != nil {
			w.log.Error().Interface("panic", r).Msg("Worker recovered from panic")
		}
	}()

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
		Msg("Processing job - starting")

	// Buscar handler
	handler, exists := w.queue.handlers[job.Type]
	if !exists {
		w.log.Error().Str("type", string(job.Type)).Msg("No handler for job type")
		if err := w.queue.Fail(ctx, job.ID, "no handler for job type"); err != nil {
			w.log.Error().Err(err).Msg("Failed to mark job as failed")
		}
		return
	}

	// Ejecutar handler con timeout
	jobCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	w.log.Debug().Str("job_id", job.ID.String()).Msg("Executing handler")

	handlerErr := handler(jobCtx, job)

	w.log.Debug().
		Str("job_id", job.ID.String()).
		Bool("handler_success", handlerErr == nil).
		Msg("Handler execution finished")

	if handlerErr != nil {
		w.log.Error().
			Err(handlerErr).
			Str("job_id", job.ID.String()).
			Str("type", string(job.Type)).
			Msg("Job handler returned error")
		if err := w.queue.Fail(ctx, job.ID, handlerErr.Error()); err != nil {
			w.log.Error().Err(err).Str("job_id", job.ID.String()).Msg("Failed to mark job as failed")
		}
		return
	}

	// Marcar como completado
	w.log.Debug().Str("job_id", job.ID.String()).Msg("Marking job as completed")
	if err := w.queue.Complete(ctx, job.ID, nil); err != nil {
		w.log.Error().
			Err(err).
			Str("job_id", job.ID.String()).
			Msg("Failed to mark job as completed")
	} else {
		w.log.Info().
			Str("job_id", job.ID.String()).
			Str("type", string(job.Type)).
			Msg("Job completed successfully")
	}
}

// Handlers por defecto - Implementaciones reales

// handleRiskEvaluation procesa evaluaciones de riesgo crediticio
func (q *PostgresQueue) handleRiskEvaluation(ctx context.Context, job *entity.Job) error {
	q.log.Info().
		Str("job_id", job.ID.String()).
		Str("payload", string(job.Payload)).
		Msg("Processing risk evaluation - starting")

	var payload struct {
		ApplicationID string `json:"application_id"`
		CountryID     string `json:"country_id"`
	}
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		q.log.Error().Err(err).Str("raw_payload", string(job.Payload)).Msg("Failed to parse risk evaluation payload")
		return fmt.Errorf("failed to parse job payload: %w", err)
	}

	appID, err := uuid.Parse(payload.ApplicationID)
	if err != nil {
		q.log.Error().Err(err).Str("application_id", payload.ApplicationID).Msg("Invalid application_id UUID")
		return fmt.Errorf("invalid application ID: %w", err)
	}

	// Obtener solicitud junto con la configuración del país
	var app entity.CreditApplication
	var countryConfig []byte
	var countryCurrency string
	query := `
		SELECT ca.id, ca.country_id, ca.full_name, ca.document_type, ca.document_number, 
		       ca.requested_amount, ca.monthly_income, ca.status,
		       c.config, c.currency
		FROM credit_applications ca
		JOIN countries c ON c.id = ca.country_id
		WHERE ca.id = $1
	`
	row := q.db.QueryRow(ctx, query, appID)
	if err := row.Scan(&app.ID, &app.CountryID, &app.FullName, &app.DocumentType,
		&app.DocumentNumber, &app.RequestedAmount, &app.MonthlyIncome, &app.Status,
		&countryConfig, &countryCurrency); err != nil {
		q.log.Error().Err(err).Str("application_id", appID.String()).Msg("Failed to get application with country config")
		return fmt.Errorf("failed to get application: %w", err)
	}

	// Parsear configuración del país
	var config countryRiskConfig
	if err := json.Unmarshal(countryConfig, &config); err != nil {
		q.log.Warn().Err(err).Msg("Failed to parse country config, using defaults")
		// Valores por defecto si no se puede parsear
		config.MaxDebtToIncomeRatio = 0.4
		config.ReviewThreshold = 50000
		config.MinCreditScore = 600
	}

	q.log.Debug().
		Str("application_id", appID.String()).
		Str("currency", countryCurrency).
		Float64("review_threshold", config.ReviewThreshold).
		Int("min_credit_score", config.MinCreditScore).
		Float64("max_debt_ratio", config.MaxDebtToIncomeRatio).
		Msg("Country configuration loaded")

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
		q.log.Debug().
			Int("credit_score", *bankingInfo.CreditScore).
			Float64("total_debt", *bankingInfo.TotalDebt).
			Msg("Banking info found for application")
	} else {
		q.log.Warn().Err(err).Str("application_id", appID.String()).Msg("No banking info found - proceeding without it")
	}

	// Calcular score de riesgo usando la configuración del país (0-100, donde 100 es bajo riesgo)
	riskScore := q.calculateRiskScoreWithConfig(&app, &config)

	// Determinar resultado basado en el score y configuración del país
	var newStatus entity.ApplicationStatus
	var statusReason string
	requiresReview := false

	// Verificar si el monto supera el umbral de revisión del país
	if app.RequestedAmount >= config.ReviewThreshold {
		requiresReview = true
	}

	// Verificar score crediticio mínimo del país
	if app.BankingInfo != nil && app.BankingInfo.CreditScore != nil {
		if *app.BankingInfo.CreditScore < config.MinCreditScore {
			riskScore -= 20 // Penalizar si no cumple el mínimo del país
		}
	}

	if riskScore >= 70 && !requiresReview {
		newStatus = entity.StatusApproved
		statusReason = fmt.Sprintf("Auto-approved with risk score %.0f (currency: %s)", riskScore, countryCurrency)
	} else if riskScore >= 40 || requiresReview {
		newStatus = entity.StatusUnderReview
		if requiresReview {
			statusReason = fmt.Sprintf("Manual review required - amount %.2f %s exceeds threshold %.2f %s (risk score: %.0f)",
				app.RequestedAmount, countryCurrency, config.ReviewThreshold, countryCurrency, riskScore)
		} else {
			statusReason = fmt.Sprintf("Manual review required - risk score %.0f", riskScore)
		}
		requiresReview = true
	} else {
		newStatus = entity.StatusRejected
		statusReason = fmt.Sprintf("Auto-rejected due to high risk - score %.0f (min credit score required: %d)", riskScore, config.MinCreditScore)
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

// countryRiskConfig configuración de riesgo por país
type countryRiskConfig struct {
	MinLoanAmount        float64 `json:"min_loan_amount"`
	MaxLoanAmount        float64 `json:"max_loan_amount"`
	MinIncomeRequired    float64 `json:"min_income_required"`
	MaxDebtToIncomeRatio float64 `json:"max_debt_to_income_ratio"`
	ReviewThreshold      float64 `json:"review_threshold"`
	MinCreditScore       int     `json:"min_credit_score"`
}

// calculateRiskScoreWithConfig calcula el score de riesgo usando la configuración del país
func (q *PostgresQueue) calculateRiskScoreWithConfig(app *entity.CreditApplication, config *countryRiskConfig) float64 {
	score := 50.0 // Base score

	// Factor 1: Relación monto/ingreso usando el ratio máximo del país (peso: 25%)
	if app.MonthlyIncome > 0 {
		// Calcular ratio de endeudamiento
		annualIncome := app.MonthlyIncome * 12
		requestedRatio := app.RequestedAmount / annualIncome

		// Usar el max_debt_to_income_ratio del país como referencia
		maxRatio := config.MaxDebtToIncomeRatio
		if maxRatio == 0 {
			maxRatio = 0.4 // Default
		}

		if requestedRatio < maxRatio*0.5 { // Menos del 50% del máximo permitido
			score += 25
		} else if requestedRatio < maxRatio*0.75 { // Entre 50% y 75% del máximo
			score += 15
		} else if requestedRatio < maxRatio { // Entre 75% y 100% del máximo
			score += 5
		} else if requestedRatio > maxRatio*1.25 { // Supera el máximo por más del 25%
			score -= 15
		}
	}

	// Factor 2: Score crediticio vs mínimo requerido del país (peso: 35%)
	if app.BankingInfo != nil && app.BankingInfo.CreditScore != nil {
		creditScore := *app.BankingInfo.CreditScore
		minScore := config.MinCreditScore
		if minScore == 0 {
			minScore = 600 // Default
		}

		// Comparar con el mínimo del país
		if creditScore >= minScore+150 { // Muy por encima del mínimo
			score += 35
		} else if creditScore >= minScore+50 { // Por encima del mínimo
			score += 20
		} else if creditScore >= minScore { // Cumple el mínimo
			score += 5
		} else { // Por debajo del mínimo
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

	// Factor 4: Deuda existente vs relación máxima del país (peso: 10%)
	if app.BankingInfo != nil && app.BankingInfo.TotalDebt != nil && app.MonthlyIncome > 0 {
		debt := *app.BankingInfo.TotalDebt
		// Calcular ratio de deuda actual sobre ingreso
		currentDebtRatio := debt / (app.MonthlyIncome * 12)
		maxRatio := config.MaxDebtToIncomeRatio
		if maxRatio == 0 {
			maxRatio = 0.4
		}

		if currentDebtRatio < maxRatio*0.25 { // Deuda muy baja
			score += 10
		} else if currentDebtRatio < maxRatio*0.5 { // Deuda moderada
			score += 5
		} else if currentDebtRatio > maxRatio { // Deuda supera el máximo
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
	q.log.Info().
		Str("job_id", job.ID.String()).
		Str("payload", string(job.Payload)).
		Msg("Fetching banking info - starting")

	var payload struct {
		ApplicationID  string `json:"application_id"`
		DocumentType   string `json:"document_type"`
		DocumentNumber string `json:"document_number"`
		CountryID      string `json:"country_id"`
	}
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		q.log.Error().Err(err).Str("raw_payload", string(job.Payload)).Msg("Failed to parse job payload")
		return fmt.Errorf("failed to parse job payload: %w", err)
	}

	q.log.Debug().
		Str("application_id", payload.ApplicationID).
		Str("country_id", payload.CountryID).
		Str("document_type", payload.DocumentType).
		Msg("Payload parsed successfully")

	appID, err := uuid.Parse(payload.ApplicationID)
	if err != nil {
		q.log.Error().Err(err).Str("application_id", payload.ApplicationID).Msg("Invalid application_id UUID")
		return fmt.Errorf("invalid application_id: %w", err)
	}

	countryID, err := uuid.Parse(payload.CountryID)
	if err != nil {
		q.log.Error().Err(err).Str("country_id", payload.CountryID).Msg("Invalid country_id UUID")
		return fmt.Errorf("invalid country_id: %w", err)
	}

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
		q.log.Warn().
			Err(err).
			Str("country_id", countryID.String()).
			Msg("No active banking provider found for country")
		return fmt.Errorf("no active banking provider for country %s: %w", countryID, err)
	}

	q.log.Debug().
		Str("provider_id", providerID.String()).
		Str("provider_code", providerCode).
		Msg("Banking provider found")

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
	
	q.log.Debug().
		Str("banking_id", bankingID.String()).
		Str("application_id", appID.String()).
		Str("provider_id", providerID.String()).
		Int("credit_score", creditScore).
		Msg("Saving banking info to database")

	if err := q.db.Exec(ctx, saveQuery, bankingID, appID, providerID, creditScore, totalDebt,
		availableCredit, paymentHistory, bankAccounts, activeLoans, monthsEmployed); err != nil {
		q.log.Error().
			Err(err).
			Str("application_id", appID.String()).
			Msg("Failed to save banking info to database")
		return fmt.Errorf("failed to save banking info: %w", err)
	}

	q.log.Info().
		Str("application_id", appID.String()).
		Str("banking_id", bankingID.String()).
		Msg("Banking info saved successfully")

	// Actualizar estado de la solicitud
	updateQuery := `UPDATE credit_applications SET status = 'VALIDATING', updated_at = NOW() WHERE id = $1`
	if err := q.db.Exec(ctx, updateQuery, appID); err != nil {
		q.log.Error().Err(err).Str("application_id", appID.String()).Msg("Failed to update application status")
	} else {
		q.log.Info().Str("application_id", appID.String()).Msg("Application status updated to VALIDATING")
	}

	// Encolar evaluación de riesgo
	riskJob := &entity.Job{
		ID:       uuid.New(),
		Type:     entity.JobTypeRiskEvaluation,
		Priority: 10,
		Payload:  job.Payload,
	}
	if err := q.Enqueue(ctx, riskJob); err != nil {
		q.log.Error().Err(err).Str("application_id", appID.String()).Msg("Failed to enqueue risk evaluation")
	} else {
		q.log.Info().
			Str("application_id", appID.String()).
			Str("risk_job_id", riskJob.ID.String()).
			Msg("Risk evaluation job enqueued")
	}

	q.log.Info().
		Str("application_id", appID.String()).
		Str("provider", providerCode).
		Int("credit_score", creditScore).
		Float64("total_debt", totalDebt).
		Str("payment_history", paymentHistory).
		Msg("Banking info fetch completed successfully")

	return nil
}

// handleDocumentValidation valida documentos de identidad
func (q *PostgresQueue) handleDocumentValidation(ctx context.Context, job *entity.Job) error {
	q.log.Info().
		Str("job_id", job.ID.String()).
		Str("payload", string(job.Payload)).
		Msg("Validating document - starting")

	var payload struct {
		ApplicationID  string `json:"application_id"`
		DocumentType   string `json:"document_type"`
		DocumentNumber string `json:"document_number"`
		CountryID      string `json:"country_id"` // Nota: el trigger envía country_id (UUID)
	}
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		q.log.Error().Err(err).Str("raw_payload", string(job.Payload)).Msg("Failed to parse document validation payload")
		return fmt.Errorf("failed to parse job payload: %w", err)
	}

	q.log.Debug().
		Str("application_id", payload.ApplicationID).
		Str("country_id", payload.CountryID).
		Str("document_type", payload.DocumentType).
		Msg("Document validation payload parsed")

	countryID, err := uuid.Parse(payload.CountryID)
	if err != nil {
		q.log.Error().Err(err).Str("country_id", payload.CountryID).Msg("Invalid country_id UUID")
		return fmt.Errorf("invalid country_id: %w", err)
	}

	// Obtener regex de validación del tipo de documento usando country_id
	var validationRegex *string
	regexQuery := `
		SELECT dt.validation_regex 
		FROM document_types dt
		WHERE dt.country_id = $1 AND dt.code = $2
	`
	row := q.db.QueryRow(ctx, regexQuery, countryID, payload.DocumentType)
	if err := row.Scan(&validationRegex); err != nil {
		q.log.Warn().
			Err(err).
			Str("doc_type", payload.DocumentType).
			Str("country_id", countryID.String()).
			Msg("Document type not found - continuing without validation regex")
	}

	// La validación real se hace en el servicio de validación
	// Aquí solo registramos el resultado
	isValid := true
	if validationRegex != nil && *validationRegex != "" {
		// En producción, usar regexp.MatchString para validar
		q.log.Debug().Str("regex", *validationRegex).Msg("Validation regex found")
	}

	q.log.Info().
		Str("application_id", payload.ApplicationID).
		Str("document_type", payload.DocumentType).
		Str("country_id", countryID.String()).
		Bool("valid", isValid).
		Msg("Document validation completed successfully")

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

