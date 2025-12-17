package handler

import (
	"net/http"
	"time"

	"github.com/fintech-multipass/backend/internal/infrastructure/database"
	"github.com/fintech-multipass/backend/internal/infrastructure/logger"
	"github.com/gin-gonic/gin"
)

// StatsHandler maneja las solicitudes de estadísticas
type StatsHandler struct {
	db  *database.PostgresDB
	log *logger.Logger
}

// NewStatsHandler crea una nueva instancia del handler
func NewStatsHandler(db *database.PostgresDB, log *logger.Logger) *StatsHandler {
	return &StatsHandler{
		db:  db,
		log: log,
	}
}

// DashboardStats representa las estadísticas del dashboard
type DashboardStats struct {
	Summary       Summary                  `json:"summary"`
	ByStatus      []StatusCount            `json:"by_status"`
	ByCountry     []CountryStats           `json:"by_country"`
	RecentTrends  RecentTrends             `json:"recent_trends"`
	RiskAnalysis  RiskAnalysis             `json:"risk_analysis"`
	TopApplicants []TopApplicant           `json:"top_applicants,omitempty"`
	LastUpdated   time.Time                `json:"last_updated"`
}

// Summary resumen general
type Summary struct {
	TotalApplications     int64   `json:"total_applications"`
	PendingReview         int64   `json:"pending_review"`
	ApprovedToday         int64   `json:"approved_today"`
	RejectedToday         int64   `json:"rejected_today"`
	TotalAmountRequested  float64 `json:"total_amount_requested"`
	TotalAmountApproved   float64 `json:"total_amount_approved"`
	AverageProcessingTime float64 `json:"average_processing_time_hours"`
	ApprovalRate          float64 `json:"approval_rate"`
}

// StatusCount conteo por estado
type StatusCount struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

// CountryStats estadísticas por país
type CountryStats struct {
	CountryCode       string  `json:"country_code"`
	CountryName       string  `json:"country_name"`
	TotalApplications int64   `json:"total_applications"`
	ApprovalRate      float64 `json:"approval_rate"`
	TotalAmount       float64 `json:"total_amount"`
	AvgRiskScore      float64 `json:"avg_risk_score"`
}

// RecentTrends tendencias recientes
type RecentTrends struct {
	Last7Days    TrendData `json:"last_7_days"`
	Last30Days   TrendData `json:"last_30_days"`
	DailyVolume  []DayData `json:"daily_volume"`
}

// TrendData datos de tendencia
type TrendData struct {
	Applications int64   `json:"applications"`
	Approved     int64   `json:"approved"`
	Rejected     int64   `json:"rejected"`
	TotalAmount  float64 `json:"total_amount"`
	ChangePercent float64 `json:"change_percent"` // vs periodo anterior
}

// DayData datos por día
type DayData struct {
	Date         string `json:"date"`
	Applications int64  `json:"applications"`
	Approved     int64  `json:"approved"`
	Rejected     int64  `json:"rejected"`
}

// RiskAnalysis análisis de riesgo
type RiskAnalysis struct {
	AvgRiskScore       float64      `json:"avg_risk_score"`
	LowRiskCount       int64        `json:"low_risk_count"`     // score >= 70
	MediumRiskCount    int64        `json:"medium_risk_count"`  // score 40-69
	HighRiskCount      int64        `json:"high_risk_count"`    // score < 40
	RiskDistribution   []RiskBucket `json:"risk_distribution"`
}

// RiskBucket distribución de riesgo
type RiskBucket struct {
	Range string `json:"range"`
	Count int64  `json:"count"`
}

// TopApplicant top solicitantes
type TopApplicant struct {
	DocumentMasked string  `json:"document_masked"`
	Applications   int64   `json:"applications"`
	TotalAmount    float64 `json:"total_amount"`
}

// GetDashboardStats obtiene las estadísticas del dashboard
// @Summary Obtener estadísticas del dashboard
// @Description Retorna estadísticas agregadas para el dashboard administrativo
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} DashboardStats
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/admin/stats [get]
func (h *StatsHandler) GetDashboardStats(c *gin.Context) {
	ctx := c.Request.Context()

	stats := DashboardStats{
		LastUpdated: time.Now(),
	}

	// Summary
	summaryQuery := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE requires_review = true AND status NOT IN ('APPROVED', 'REJECTED', 'CANCELLED', 'EXPIRED', 'DISBURSED')) as pending_review,
			COUNT(*) FILTER (WHERE status = 'APPROVED' AND DATE(processed_at) = CURRENT_DATE) as approved_today,
			COUNT(*) FILTER (WHERE status = 'REJECTED' AND DATE(processed_at) = CURRENT_DATE) as rejected_today,
			COALESCE(SUM(requested_amount), 0) as total_requested,
			COALESCE(SUM(requested_amount) FILTER (WHERE status = 'APPROVED'), 0) as total_approved,
			COALESCE(AVG(EXTRACT(EPOCH FROM (processed_at - created_at))/3600) FILTER (WHERE processed_at IS NOT NULL), 0) as avg_processing_hours,
			CASE 
				WHEN COUNT(*) FILTER (WHERE status IN ('APPROVED', 'REJECTED')) > 0 
				THEN ROUND(COUNT(*) FILTER (WHERE status = 'APPROVED')::numeric / COUNT(*) FILTER (WHERE status IN ('APPROVED', 'REJECTED')) * 100, 2)
				ELSE 0 
			END as approval_rate
		FROM credit_applications
	`

	row := h.db.QueryRow(ctx, summaryQuery)
	if err := row.Scan(
		&stats.Summary.TotalApplications,
		&stats.Summary.PendingReview,
		&stats.Summary.ApprovedToday,
		&stats.Summary.RejectedToday,
		&stats.Summary.TotalAmountRequested,
		&stats.Summary.TotalAmountApproved,
		&stats.Summary.AverageProcessingTime,
		&stats.Summary.ApprovalRate,
	); err != nil {
		h.log.Error().Err(err).Msg("Failed to get summary stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get statistics"})
		return
	}

	// By Status
	statusQuery := `
		SELECT status, COUNT(*) as count
		FROM credit_applications
		GROUP BY status
		ORDER BY count DESC
	`
	rows, err := h.db.Query(ctx, statusQuery)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var sc StatusCount
			if err := rows.Scan(&sc.Status, &sc.Count); err == nil {
				stats.ByStatus = append(stats.ByStatus, sc)
			}
		}
	}

	// By Country
	countryQuery := `
		SELECT 
			c.code,
			c.name,
			COUNT(ca.id) as total_apps,
			CASE 
				WHEN COUNT(ca.id) FILTER (WHERE ca.status IN ('APPROVED', 'REJECTED')) > 0 
				THEN ROUND(COUNT(ca.id) FILTER (WHERE ca.status = 'APPROVED')::numeric / 
				     COUNT(ca.id) FILTER (WHERE ca.status IN ('APPROVED', 'REJECTED')) * 100, 2)
				ELSE 0 
			END as approval_rate,
			COALESCE(SUM(ca.requested_amount), 0) as total_amount,
			COALESCE(AVG(ca.risk_score), 0) as avg_risk
		FROM countries c
		LEFT JOIN credit_applications ca ON c.id = ca.country_id
		WHERE c.is_active = true
		GROUP BY c.id, c.code, c.name
		ORDER BY total_apps DESC
	`
	rows, err = h.db.Query(ctx, countryQuery)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var cs CountryStats
			if err := rows.Scan(&cs.CountryCode, &cs.CountryName, &cs.TotalApplications,
				&cs.ApprovalRate, &cs.TotalAmount, &cs.AvgRiskScore); err == nil {
				stats.ByCountry = append(stats.ByCountry, cs)
			}
		}
	}

	// Recent Trends - Last 7 days
	trend7Query := `
		SELECT 
			COUNT(*) as apps,
			COUNT(*) FILTER (WHERE status = 'APPROVED') as approved,
			COUNT(*) FILTER (WHERE status = 'REJECTED') as rejected,
			COALESCE(SUM(requested_amount), 0) as total_amount
		FROM credit_applications
		WHERE created_at >= NOW() - INTERVAL '7 days'
	`
	row = h.db.QueryRow(ctx, trend7Query)
	row.Scan(&stats.RecentTrends.Last7Days.Applications, &stats.RecentTrends.Last7Days.Approved,
		&stats.RecentTrends.Last7Days.Rejected, &stats.RecentTrends.Last7Days.TotalAmount)

	// Recent Trends - Last 30 days
	trend30Query := `
		SELECT 
			COUNT(*) as apps,
			COUNT(*) FILTER (WHERE status = 'APPROVED') as approved,
			COUNT(*) FILTER (WHERE status = 'REJECTED') as rejected,
			COALESCE(SUM(requested_amount), 0) as total_amount
		FROM credit_applications
		WHERE created_at >= NOW() - INTERVAL '30 days'
	`
	row = h.db.QueryRow(ctx, trend30Query)
	row.Scan(&stats.RecentTrends.Last30Days.Applications, &stats.RecentTrends.Last30Days.Approved,
		&stats.RecentTrends.Last30Days.Rejected, &stats.RecentTrends.Last30Days.TotalAmount)

	// Daily volume last 7 days
	dailyQuery := `
		SELECT 
			DATE(created_at) as day,
			COUNT(*) as apps,
			COUNT(*) FILTER (WHERE status = 'APPROVED') as approved,
			COUNT(*) FILTER (WHERE status = 'REJECTED') as rejected
		FROM credit_applications
		WHERE created_at >= NOW() - INTERVAL '7 days'
		GROUP BY DATE(created_at)
		ORDER BY day DESC
	`
	rows, err = h.db.Query(ctx, dailyQuery)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var dd DayData
			var day time.Time
			if err := rows.Scan(&day, &dd.Applications, &dd.Approved, &dd.Rejected); err == nil {
				dd.Date = day.Format("2006-01-02")
				stats.RecentTrends.DailyVolume = append(stats.RecentTrends.DailyVolume, dd)
			}
		}
	}

	// Risk Analysis
	riskQuery := `
		SELECT 
			COALESCE(AVG(risk_score), 0) as avg_score,
			COUNT(*) FILTER (WHERE risk_score >= 70) as low_risk,
			COUNT(*) FILTER (WHERE risk_score >= 40 AND risk_score < 70) as medium_risk,
			COUNT(*) FILTER (WHERE risk_score < 40 AND risk_score IS NOT NULL) as high_risk
		FROM credit_applications
		WHERE risk_score IS NOT NULL
	`
	row = h.db.QueryRow(ctx, riskQuery)
	row.Scan(&stats.RiskAnalysis.AvgRiskScore, &stats.RiskAnalysis.LowRiskCount,
		&stats.RiskAnalysis.MediumRiskCount, &stats.RiskAnalysis.HighRiskCount)

	// Risk Distribution
	riskDistQuery := `
		SELECT 
			CASE 
				WHEN risk_score >= 90 THEN '90-100'
				WHEN risk_score >= 80 THEN '80-89'
				WHEN risk_score >= 70 THEN '70-79'
				WHEN risk_score >= 60 THEN '60-69'
				WHEN risk_score >= 50 THEN '50-59'
				WHEN risk_score >= 40 THEN '40-49'
				WHEN risk_score >= 30 THEN '30-39'
				WHEN risk_score >= 20 THEN '20-29'
				ELSE '0-19'
			END as range,
			COUNT(*) as count
		FROM credit_applications
		WHERE risk_score IS NOT NULL
		GROUP BY range
		ORDER BY range DESC
	`
	rows, err = h.db.Query(ctx, riskDistQuery)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var rb RiskBucket
			if err := rows.Scan(&rb.Range, &rb.Count); err == nil {
				stats.RiskAnalysis.RiskDistribution = append(stats.RiskAnalysis.RiskDistribution, rb)
			}
		}
	}

	c.JSON(http.StatusOK, stats)
}

// GetCountryStats obtiene estadísticas detalladas de un país específico
// @Summary Obtener estadísticas de un país
// @Description Retorna estadísticas detalladas para un país específico
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param code path string true "Código del país"
// @Success 200 {object} CountryStats
// @Failure 404 {object} map[string]string
// @Router /api/v1/admin/stats/country/{code} [get]
func (h *StatsHandler) GetCountryStats(c *gin.Context) {
	countryCode := c.Param("code")
	ctx := c.Request.Context()

	query := `
		SELECT 
			c.code,
			c.name,
			COUNT(ca.id) as total_apps,
			CASE 
				WHEN COUNT(ca.id) FILTER (WHERE ca.status IN ('APPROVED', 'REJECTED')) > 0 
				THEN ROUND(COUNT(ca.id) FILTER (WHERE ca.status = 'APPROVED')::numeric / 
				     COUNT(ca.id) FILTER (WHERE ca.status IN ('APPROVED', 'REJECTED')) * 100, 2)
				ELSE 0 
			END as approval_rate,
			COALESCE(SUM(ca.requested_amount), 0) as total_amount,
			COALESCE(AVG(ca.risk_score), 0) as avg_risk
		FROM countries c
		LEFT JOIN credit_applications ca ON c.id = ca.country_id
		WHERE c.code = $1
		GROUP BY c.id, c.code, c.name
	`

	var cs CountryStats
	row := h.db.QueryRow(ctx, query, countryCode)
	if err := row.Scan(&cs.CountryCode, &cs.CountryName, &cs.TotalApplications,
		&cs.ApprovalRate, &cs.TotalAmount, &cs.AvgRiskScore); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Country not found"})
		return
	}

	c.JSON(http.StatusOK, cs)
}

