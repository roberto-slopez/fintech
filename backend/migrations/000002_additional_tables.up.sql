-- Migración 002: Tablas adicionales para webhooks, auditoría y notificaciones

-- Tabla de endpoints de webhooks
CREATE TABLE IF NOT EXISTS webhook_endpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    country_id UUID REFERENCES countries(id) ON DELETE CASCADE,
    url VARCHAR(500) NOT NULL,
    secret VARCHAR(255),
    event_types VARCHAR(100)[] NOT NULL DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    max_retries INT DEFAULT 3,
    retry_delay_seconds INT DEFAULT 60,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Tabla de eventos de webhooks
CREATE TABLE IF NOT EXISTS webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    application_id UUID REFERENCES credit_applications(id) ON DELETE SET NULL,
    country_code VARCHAR(10),
    data JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Tabla de entregas de webhooks
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id UUID REFERENCES webhook_endpoints(id) ON DELETE CASCADE,
    event_id UUID REFERENCES webhook_events(id) ON DELETE CASCADE,
    status VARCHAR(20) DEFAULT 'PENDING', -- PENDING, SENT, FAILED
    http_status INT,
    response_body TEXT,
    attempts INT DEFAULT 0,
    last_attempt TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Tabla de logs de auditoría
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,
    actor_id UUID,
    actor_type VARCHAR(20), -- USER, SYSTEM, WEBHOOK
    old_values JSONB,
    new_values JSONB,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Tabla de información bancaria
CREATE TABLE IF NOT EXISTS banking_info (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID UNIQUE REFERENCES credit_applications(id) ON DELETE CASCADE,
    provider_id UUID REFERENCES banking_providers(id),
    credit_score INT,
    total_debt DECIMAL(15,2),
    available_credit DECIMAL(15,2),
    payment_history VARCHAR(20),
    bank_accounts INT DEFAULT 0,
    active_loans INT DEFAULT 0,
    months_employed INT,
    raw_response JSONB,
    retrieved_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE
);

-- Tabla de transiciones de estado
CREATE TABLE IF NOT EXISTS state_transitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID REFERENCES credit_applications(id) ON DELETE CASCADE,
    from_status VARCHAR(30),
    to_status VARCHAR(30) NOT NULL,
    reason TEXT,
    triggered_by VARCHAR(20) NOT NULL, -- SYSTEM, USER, WEBHOOK
    triggered_by_id UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Tabla de notificaciones enviadas
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(20) NOT NULL, -- EMAIL, SMS, PUSH
    recipient VARCHAR(255) NOT NULL,
    subject VARCHAR(500),
    template VARCHAR(100),
    data JSONB,
    status VARCHAR(20) DEFAULT 'PENDING', -- PENDING, SENT, FAILED
    message_id VARCHAR(255),
    error_message TEXT,
    application_id UUID REFERENCES credit_applications(id) ON DELETE SET NULL,
    sent_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Índices para rendimiento
CREATE INDEX IF NOT EXISTS idx_webhook_events_type ON webhook_events(event_type);
CREATE INDEX IF NOT EXISTS idx_webhook_events_app ON webhook_events(application_id);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_status ON webhook_deliveries(status);
CREATE INDEX IF NOT EXISTS idx_audit_logs_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor ON audit_logs(actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_banking_info_app ON banking_info(application_id);
CREATE INDEX IF NOT EXISTS idx_state_transitions_app ON state_transitions(application_id);
CREATE INDEX IF NOT EXISTS idx_notifications_app ON notifications(application_id);
CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status);

-- Agregar columna risk_score a credit_applications si no existe
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='credit_applications' AND column_name='risk_score') THEN
        ALTER TABLE credit_applications ADD COLUMN risk_score DECIMAL(5,2);
    END IF;
END $$;

-- Datos de prueba para webhook endpoints
INSERT INTO webhook_endpoints (country_id, url, secret, event_types, is_active) 
SELECT c.id, 'https://webhook.example.com/fintech/' || c.code, 'secret_' || c.code, 
       ARRAY['application.created', 'application.approved', 'application.rejected'],
       true
FROM countries c
WHERE c.is_active = true
ON CONFLICT DO NOTHING;


