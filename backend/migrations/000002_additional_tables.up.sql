-- Migración 002: Tablas adicionales para webhooks salientes

-- Tabla de endpoints de webhooks (para webhooks salientes)
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

-- Tabla de entregas de webhooks salientes
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id UUID REFERENCES webhook_endpoints(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) DEFAULT 'PENDING', -- PENDING, SENT, FAILED
    http_status INT,
    response_body TEXT,
    attempts INT DEFAULT 0,
    last_attempt TIMESTAMP WITH TIME ZONE,
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
CREATE INDEX IF NOT EXISTS idx_webhook_endpoints_country ON webhook_endpoints(country_id);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_status ON webhook_deliveries(status);
CREATE INDEX IF NOT EXISTS idx_notifications_app ON notifications(application_id);
CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status);

-- Trigger para updated_at en webhook_endpoints
DROP TRIGGER IF EXISTS update_webhook_endpoints_updated_at ON webhook_endpoints;
CREATE TRIGGER update_webhook_endpoints_updated_at BEFORE UPDATE ON webhook_endpoints
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Datos de prueba para webhook endpoints
INSERT INTO webhook_endpoints (country_id, url, secret, event_types, is_active) 
SELECT c.id, 'https://webhook.example.com/fintech/' || c.code, 'secret_' || c.code, 
       ARRAY['application.created', 'application.approved', 'application.rejected'],
       true
FROM countries c
WHERE c.is_active = true
ON CONFLICT DO NOTHING;
