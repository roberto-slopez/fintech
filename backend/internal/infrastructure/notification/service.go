package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"time"

	"github.com/fintech-multipass/backend/internal/domain/entity"
	"github.com/fintech-multipass/backend/internal/infrastructure/config"
	"github.com/fintech-multipass/backend/internal/infrastructure/logger"
)

// NotificationService servicio de notificaciones
type NotificationService struct {
	cfg *config.Config
	log *logger.Logger
}

// NewNotificationService crea una nueva instancia del servicio
func NewNotificationService(cfg *config.Config, log *logger.Logger) *NotificationService {
	return &NotificationService{
		cfg: cfg,
		log: log,
	}
}

// NotificationType tipos de notificaciones
type NotificationType string

const (
	NotificationTypeEmail NotificationType = "EMAIL"
	NotificationTypeSMS   NotificationType = "SMS"
	NotificationTypePush  NotificationType = "PUSH"
)

// NotificationRequest solicitud de notificación
type NotificationRequest struct {
	Type        NotificationType       `json:"type"`
	Recipient   string                 `json:"recipient"`
	Subject     string                 `json:"subject,omitempty"`
	Template    string                 `json:"template"`
	Data        map[string]interface{} `json:"data"`
	ApplicationID *string              `json:"application_id,omitempty"`
	CountryCode string                 `json:"country_code,omitempty"`
}

// NotificationResult resultado del envío de notificación
type NotificationResult struct {
	Success   bool      `json:"success"`
	MessageID string    `json:"message_id,omitempty"`
	Error     string    `json:"error,omitempty"`
	SentAt    time.Time `json:"sent_at"`
}

// SendNotification envía una notificación
func (s *NotificationService) SendNotification(ctx context.Context, req NotificationRequest) (*NotificationResult, error) {
	switch req.Type {
	case NotificationTypeEmail:
		return s.sendEmail(ctx, req)
	case NotificationTypeSMS:
		return s.sendSMS(ctx, req)
	case NotificationTypePush:
		return s.sendPush(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported notification type: %s", req.Type)
	}
}

// sendEmail envía un email
func (s *NotificationService) sendEmail(ctx context.Context, req NotificationRequest) (*NotificationResult, error) {
	result := &NotificationResult{SentAt: time.Now()}

	// Renderizar template
	body, err := s.renderTemplate(req.Template, req.Data)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}

	// En esta implementación, usamos modo simulado
	// Para producción, se debería agregar configuración SMTP al config.Config
	// y usar smtp.SendMail con los datos reales
	s.log.Info().
		Str("to", req.Recipient).
		Str("subject", req.Subject).
		Str("template", req.Template).
		Int("body_length", len(body)).
		Msg("Email notification (simulated)")
	result.Success = true
	result.MessageID = fmt.Sprintf("sim-%d", time.Now().UnixNano())

	return result, nil
}

// sendSMS envía un SMS (simulado)
func (s *NotificationService) sendSMS(ctx context.Context, req NotificationRequest) (*NotificationResult, error) {
	result := &NotificationResult{SentAt: time.Now()}

	body, err := s.renderTemplate(req.Template, req.Data)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}

	// En producción, integrar con Twilio, SNS, etc.
	s.log.Info().
		Str("to", req.Recipient).
		Str("body", body).
		Msg("SMS notification (simulated)")

	result.Success = true
	result.MessageID = fmt.Sprintf("sms-%d", time.Now().UnixNano())
	return result, nil
}

// sendPush envía una notificación push (simulada)
func (s *NotificationService) sendPush(ctx context.Context, req NotificationRequest) (*NotificationResult, error) {
	result := &NotificationResult{SentAt: time.Now()}

	body, err := s.renderTemplate(req.Template, req.Data)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}

	// En producción, integrar con Firebase, APNs, etc.
	s.log.Info().
		Str("to", req.Recipient).
		Str("body", body).
		Msg("Push notification (simulated)")

	result.Success = true
	result.MessageID = fmt.Sprintf("push-%d", time.Now().UnixNano())
	return result, nil
}

// renderTemplate renderiza un template de notificación
func (s *NotificationService) renderTemplate(templateName string, data map[string]interface{}) (string, error) {
	// Templates predefinidos
	templates := map[string]string{
		"application_received": `
			<h2>Solicitud Recibida</h2>
			<p>Hola {{.full_name}},</p>
			<p>Hemos recibido tu solicitud de crédito por {{.currency}}{{.amount}}.</p>
			<p>Número de referencia: <strong>{{.reference}}</strong></p>
			<p>Te notificaremos cuando tengamos una respuesta.</p>
		`,
		"application_approved": `
			<h2>¡Solicitud Aprobada!</h2>
			<p>Hola {{.full_name}},</p>
			<p>Nos complace informarte que tu solicitud de crédito ha sido <strong>APROBADA</strong>.</p>
			<p>Monto aprobado: {{.currency}}{{.amount}}</p>
			<p>Referencia: {{.reference}}</p>
		`,
		"application_rejected": `
			<h2>Solicitud No Aprobada</h2>
			<p>Hola {{.full_name}},</p>
			<p>Lamentamos informarte que tu solicitud de crédito no ha sido aprobada en esta ocasión.</p>
			<p>Referencia: {{.reference}}</p>
			<p>Motivo: {{.reason}}</p>
		`,
		"application_pending_review": `
			<h2>Solicitud en Revisión</h2>
			<p>Hola {{.full_name}},</p>
			<p>Tu solicitud de crédito está siendo revisada por nuestro equipo.</p>
			<p>Te contactaremos pronto con más información.</p>
			<p>Referencia: {{.reference}}</p>
		`,
		"sms_status_update": `Tu solicitud {{.reference}} está {{.status}}. Revisa tu email para más detalles.`,
	}

	templateStr, exists := templates[templateName]
	if !exists {
		return "", fmt.Errorf("template not found: %s", templateName)
	}

	tmpl, err := template.New(templateName).Parse(templateStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// SendApplicationStatusNotification envía notificación de cambio de estado
func (s *NotificationService) SendApplicationStatusNotification(ctx context.Context, app *entity.CreditApplication, newStatus entity.ApplicationStatus) error {
	var templateName string
	var subject string

	switch newStatus {
	case entity.StatusApproved:
		templateName = "application_approved"
		subject = "¡Tu solicitud ha sido aprobada!"
	case entity.StatusRejected:
		templateName = "application_rejected"
		subject = "Actualización de tu solicitud"
	case entity.StatusUnderReview:
		templateName = "application_pending_review"
		subject = "Tu solicitud está en revisión"
	default:
		return nil // No enviar notificación para otros estados
	}

	data := map[string]interface{}{
		"full_name": app.FullName,
		"amount":    fmt.Sprintf("%.2f", app.RequestedAmount),
		"currency":  "€", // TODO: obtener del país
		"reference": app.ID.String()[:8],
		"status":    string(newStatus),
		"reason":    app.StatusReason,
	}

	// Enviar email
	emailReq := NotificationRequest{
		Type:      NotificationTypeEmail,
		Recipient: app.Email,
		Subject:   subject,
		Template:  templateName,
		Data:      data,
	}

	if _, err := s.SendNotification(ctx, emailReq); err != nil {
		s.log.Error().Err(err).Msg("Failed to send email notification")
	}

	// Enviar SMS si hay teléfono
	if app.Phone != "" {
		smsReq := NotificationRequest{
			Type:      NotificationTypeSMS,
			Recipient: app.Phone,
			Template:  "sms_status_update",
			Data:      data,
		}
		if _, err := s.SendNotification(ctx, smsReq); err != nil {
			s.log.Error().Err(err).Msg("Failed to send SMS notification")
		}
	}

	return nil
}

// NotificationFromJob crea una notificación desde un job
func (s *NotificationService) NotificationFromJob(ctx context.Context, job *entity.Job) error {
	var req NotificationRequest
	if err := json.Unmarshal(job.Payload, &req); err != nil {
		return fmt.Errorf("failed to parse notification job payload: %w", err)
	}

	_, err := s.SendNotification(ctx, req)
	return err
}

