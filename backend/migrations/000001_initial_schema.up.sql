-- =====================================================
-- Fintech Multipaís - Esquema Inicial
-- Diseñado para escalar a millones de registros
-- =====================================================

-- Extensiones necesarias
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm"; -- Para búsquedas de texto eficientes

-- =====================================================
-- TABLA: countries
-- Países donde opera la fintech (configurables dinámicamente)
-- =====================================================
CREATE TABLE countries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(3) NOT NULL UNIQUE,  -- ISO 3166-1 alpha-2/3
    name VARCHAR(100) NOT NULL,
    currency VARCHAR(3) NOT NULL,      -- ISO 4217
    timezone VARCHAR(50) NOT NULL DEFAULT 'UTC',
    is_active BOOLEAN NOT NULL DEFAULT true,
    
    -- Configuración específica del país (JSON para flexibilidad)
    config JSONB NOT NULL DEFAULT '{
        "min_loan_amount": 1000,
        "max_loan_amount": 100000,
        "min_income_required": 500,
        "max_debt_to_income_ratio": 0.4,
        "review_threshold": 50000,
        "min_credit_score": 600
    }'::jsonb,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Índices para countries
CREATE INDEX idx_countries_code ON countries(code);
CREATE INDEX idx_countries_active ON countries(is_active) WHERE is_active = true;

-- =====================================================
-- TABLA: document_types
-- Tipos de documentos válidos por país
-- =====================================================
CREATE TABLE document_types (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    country_id UUID NOT NULL REFERENCES countries(id) ON DELETE CASCADE,
    code VARCHAR(20) NOT NULL,         -- DNI, NIF, CURP, CPF, CC, CF
    name VARCHAR(100) NOT NULL,
    validation_regex VARCHAR(255),     -- Regex para validación
    is_required BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(country_id, code)
);

CREATE INDEX idx_document_types_country ON document_types(country_id);

-- =====================================================
-- TABLA: country_rules
-- Reglas de validación configurables por país
-- =====================================================
CREATE TABLE country_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    country_id UUID NOT NULL REFERENCES countries(id) ON DELETE CASCADE,
    rule_type VARCHAR(50) NOT NULL,    -- DOCUMENT_VALIDATION, INCOME_CHECK, DEBT_RATIO, etc.
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    priority INT NOT NULL DEFAULT 0,   -- Orden de evaluación (mayor = primero)
    
    -- Configuración flexible de la regla
    config JSONB NOT NULL DEFAULT '{}',
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_country_rules_country ON country_rules(country_id);
CREATE INDEX idx_country_rules_active ON country_rules(country_id, is_active) WHERE is_active = true;
CREATE INDEX idx_country_rules_type ON country_rules(rule_type);

-- =====================================================
-- TABLA: banking_providers
-- Proveedores bancarios por país
-- =====================================================
CREATE TABLE banking_providers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    country_id UUID NOT NULL REFERENCES countries(id) ON DELETE CASCADE,
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(30) NOT NULL,         -- CREDIT_BUREAU, BANK_API, OPEN_BANKING, AGGREGATOR
    is_active BOOLEAN NOT NULL DEFAULT true,
    priority INT NOT NULL DEFAULT 0,   -- Si hay múltiples proveedores
    
    -- Configuración del proveedor
    config JSONB NOT NULL DEFAULT '{
        "base_url": "",
        "timeout_seconds": 30,
        "retry_attempts": 3,
        "retry_delay_ms": 1000,
        "rate_limit_per_min": 100,
        "cache_ttl_minutes": 60,
        "auth_type": "API_KEY"
    }',
    
    -- Credenciales encriptadas
    credentials JSONB,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_banking_providers_country ON banking_providers(country_id);
CREATE INDEX idx_banking_providers_active ON banking_providers(country_id, is_active) WHERE is_active = true;

-- =====================================================
-- TABLA: users
-- Usuarios del sistema
-- =====================================================
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(200) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'VIEWER',  -- ADMIN, ANALYST, OPERATOR, VIEWER
    country_ids UUID[] DEFAULT '{}',             -- Países a los que tiene acceso
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_active ON users(is_active) WHERE is_active = true;

-- =====================================================
-- TABLA: credit_applications
-- Solicitudes de crédito (tabla principal)
-- Diseñada para millones de registros con particionamiento
-- =====================================================
CREATE TABLE credit_applications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    country_id UUID NOT NULL REFERENCES countries(id),
    
    -- Datos del solicitante (PII - manejar con cuidado)
    full_name VARCHAR(200) NOT NULL,
    document_type VARCHAR(20) NOT NULL,
    document_number VARCHAR(50) NOT NULL,
    email VARCHAR(255),
    phone VARCHAR(30),
    
    -- Datos financieros
    requested_amount DECIMAL(15, 2) NOT NULL,
    monthly_income DECIMAL(15, 2) NOT NULL,
    
    -- Estado y flujo
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
    status_reason TEXT,
    requires_review BOOLEAN NOT NULL DEFAULT false,
    
    -- Resultados de validación
    validation_results JSONB DEFAULT '[]',
    risk_score DECIMAL(5, 2),
    
    -- Fechas
    application_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Metadatos de auditoría
    created_by_ip INET,
    user_agent TEXT
);

-- Índices para credit_applications (optimizados para consultas frecuentes)
-- Índice compuesto para filtrado por país y estado (muy frecuente)
CREATE INDEX idx_applications_country_status ON credit_applications(country_id, status);

-- Índice para búsqueda por documento (frecuente en validaciones)
CREATE INDEX idx_applications_document ON credit_applications(country_id, document_number);

-- Índice para ordenamiento por fecha (listados paginados)
CREATE INDEX idx_applications_created ON credit_applications(created_at DESC);

-- Índice para solicitudes que requieren revisión
CREATE INDEX idx_applications_review ON credit_applications(requires_review, status) 
    WHERE requires_review = true;

-- Índice para búsqueda de texto en nombre (usando trigrams)
CREATE INDEX idx_applications_name_trgm ON credit_applications 
    USING GIN (full_name gin_trgm_ops);

-- Índice para filtrado por rango de montos
CREATE INDEX idx_applications_amount ON credit_applications(requested_amount);

-- Índice para filtrado por fecha de aplicación
CREATE INDEX idx_applications_date ON credit_applications(application_date DESC);

-- =====================================================
-- TABLA: banking_info
-- Información bancaria obtenida de proveedores
-- =====================================================
CREATE TABLE banking_info (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    application_id UUID NOT NULL REFERENCES credit_applications(id) ON DELETE CASCADE,
    provider_id UUID NOT NULL REFERENCES banking_providers(id),
    
    -- Datos obtenidos (normalizados)
    credit_score INT,
    total_debt DECIMAL(15, 2),
    available_credit DECIMAL(15, 2),
    payment_history VARCHAR(20),      -- GOOD, REGULAR, BAD
    bank_accounts INT DEFAULT 0,
    active_loans INT DEFAULT 0,
    months_employed INT,
    
    -- Datos crudos del proveedor (para auditoría)
    raw_response JSONB,
    
    retrieved_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX idx_banking_info_application ON banking_info(application_id);
CREATE INDEX idx_banking_info_provider ON banking_info(provider_id);

-- =====================================================
-- TABLA: state_transitions
-- Historial de cambios de estado
-- =====================================================
CREATE TABLE state_transitions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    application_id UUID NOT NULL REFERENCES credit_applications(id) ON DELETE CASCADE,
    from_status VARCHAR(30) NOT NULL,
    to_status VARCHAR(30) NOT NULL,
    reason TEXT,
    triggered_by VARCHAR(20) NOT NULL,  -- SYSTEM, USER, WEBHOOK
    triggered_by_id UUID,               -- ID del usuario si aplica
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Índice para consultar historial de una aplicación
CREATE INDEX idx_state_transitions_app ON state_transitions(application_id, created_at DESC);

-- =====================================================
-- TABLA: jobs_queue
-- Cola de trabajos para procesamiento asíncrono
-- =====================================================
CREATE TABLE jobs_queue (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    priority INT NOT NULL DEFAULT 0,
    payload JSONB NOT NULL,
    result JSONB,
    error_message TEXT,
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,
    worker_id VARCHAR(100),           -- ID del worker que procesa el job
    scheduled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Índice para obtener trabajos pendientes (usado por workers)
CREATE INDEX idx_jobs_pending ON jobs_queue(status, priority DESC, scheduled_at ASC) 
    WHERE status IN ('PENDING', 'RETRYING');

-- Índice para trabajos por tipo
CREATE INDEX idx_jobs_type ON jobs_queue(type, status);

-- Índice para limpieza de trabajos completados
CREATE INDEX idx_jobs_completed ON jobs_queue(completed_at) 
    WHERE status IN ('COMPLETED', 'FAILED');

-- =====================================================
-- TABLA: audit_logs
-- Registros de auditoría
-- =====================================================
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    action VARCHAR(30) NOT NULL,
    actor_type VARCHAR(20) NOT NULL,   -- USER, SYSTEM, WEBHOOK
    actor_id UUID,
    old_values JSONB,
    new_values JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Índice para consultar auditoría de una entidad
CREATE INDEX idx_audit_entity ON audit_logs(entity_type, entity_id, created_at DESC);

-- Índice para consultar por actor
CREATE INDEX idx_audit_actor ON audit_logs(actor_id, created_at DESC) WHERE actor_id IS NOT NULL;

-- =====================================================
-- TABLA: webhook_events
-- Eventos de webhooks entrantes
-- =====================================================
CREATE TABLE webhook_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source VARCHAR(100) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    signature VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'RECEIVED',
    error_message TEXT,
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_events_status ON webhook_events(status, created_at);
CREATE INDEX idx_webhook_events_source ON webhook_events(source, event_type);

-- =====================================================
-- FUNCIONES Y TRIGGERS
-- =====================================================

-- Función para actualizar updated_at automáticamente
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers para updated_at
CREATE TRIGGER update_countries_updated_at BEFORE UPDATE ON countries
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_country_rules_updated_at BEFORE UPDATE ON country_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_banking_providers_updated_at BEFORE UPDATE ON banking_providers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_credit_applications_updated_at BEFORE UPDATE ON credit_applications
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_jobs_queue_updated_at BEFORE UPDATE ON jobs_queue
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =====================================================
-- TRIGGER: Crear job de procesamiento al insertar solicitud
-- =====================================================
CREATE OR REPLACE FUNCTION on_application_created()
RETURNS TRIGGER AS $$
BEGIN
    -- Crear job de validación de documento
    INSERT INTO jobs_queue (type, priority, payload)
    VALUES (
        'DOCUMENT_VALIDATION',
        10,
        jsonb_build_object(
            'application_id', NEW.id,
            'country_id', NEW.country_id,
            'document_type', NEW.document_type,
            'document_number', NEW.document_number
        )
    );
    
    -- Crear job de obtención de información bancaria
    INSERT INTO jobs_queue (type, priority, payload)
    VALUES (
        'BANKING_INFO_FETCH',
        8,
        jsonb_build_object(
            'application_id', NEW.id,
            'country_id', NEW.country_id,
            'document_type', NEW.document_type,
            'document_number', NEW.document_number
        )
    );
    
    -- Crear registro de auditoría
    INSERT INTO audit_logs (entity_type, entity_id, action, actor_type, new_values)
    VALUES (
        'APPLICATION',
        NEW.id,
        'CREATE',
        'SYSTEM',
        to_jsonb(NEW)
    );
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_application_created
    AFTER INSERT ON credit_applications
    FOR EACH ROW EXECUTE FUNCTION on_application_created();

-- =====================================================
-- TRIGGER: Crear transición de estado y job al cambiar estado
-- =====================================================
CREATE OR REPLACE FUNCTION on_application_status_changed()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.status IS DISTINCT FROM NEW.status THEN
        -- Registrar transición de estado
        INSERT INTO state_transitions (application_id, from_status, to_status, triggered_by)
        VALUES (NEW.id, OLD.status, NEW.status, 'SYSTEM');
        
        -- Crear job de notificación
        INSERT INTO jobs_queue (type, priority, payload)
        VALUES (
            'NOTIFICATION',
            5,
            jsonb_build_object(
                'application_id', NEW.id,
                'old_status', OLD.status,
                'new_status', NEW.status,
                'email', NEW.email
            )
        );
        
        -- Crear registro de auditoría
        INSERT INTO audit_logs (entity_type, entity_id, action, actor_type, old_values, new_values)
        VALUES (
            'APPLICATION',
            NEW.id,
            'STATUS_CHANGE',
            'SYSTEM',
            jsonb_build_object('status', OLD.status),
            jsonb_build_object('status', NEW.status, 'status_reason', NEW.status_reason)
        );
        
        -- Si se aprueba, crear job de evaluación de riesgo final
        IF NEW.status = 'APPROVED' THEN
            INSERT INTO jobs_queue (type, priority, payload)
            VALUES (
                'RISK_EVALUATION',
                10,
                jsonb_build_object('application_id', NEW.id, 'country_id', NEW.country_id)
            );
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_application_status_changed
    AFTER UPDATE ON credit_applications
    FOR EACH ROW EXECUTE FUNCTION on_application_status_changed();

-- =====================================================
-- FUNCIÓN: Notificar a listeners (para WebSocket)
-- =====================================================
CREATE OR REPLACE FUNCTION notify_application_change()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM pg_notify(
        'application_changes',
        json_build_object(
            'operation', TG_OP,
            'application_id', COALESCE(NEW.id, OLD.id),
            'country_id', COALESCE(NEW.country_id, OLD.country_id),
            'status', NEW.status,
            'timestamp', NOW()
        )::text
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_notify_application_change
    AFTER INSERT OR UPDATE ON credit_applications
    FOR EACH ROW EXECUTE FUNCTION notify_application_change();

-- =====================================================
-- DATOS INICIALES: Países
-- =====================================================
INSERT INTO countries (code, name, currency, timezone, config) VALUES
('ES', 'España', 'EUR', 'Europe/Madrid', '{
    "min_loan_amount": 1000,
    "max_loan_amount": 100000,
    "min_income_required": 800,
    "max_debt_to_income_ratio": 0.35,
    "review_threshold": 30000,
    "min_credit_score": 650
}'::jsonb),
('MX', 'México', 'MXN', 'America/Mexico_City', '{
    "min_loan_amount": 5000,
    "max_loan_amount": 500000,
    "min_income_required": 5000,
    "max_debt_to_income_ratio": 0.40,
    "review_threshold": 200000,
    "min_credit_score": 600
}'::jsonb),
('CO', 'Colombia', 'COP', 'America/Bogota', '{
    "min_loan_amount": 500000,
    "max_loan_amount": 50000000,
    "min_income_required": 1000000,
    "max_debt_to_income_ratio": 0.45,
    "review_threshold": 20000000,
    "min_credit_score": 550
}'::jsonb),
('BR', 'Brasil', 'BRL', 'America/Sao_Paulo', '{
    "min_loan_amount": 1000,
    "max_loan_amount": 200000,
    "min_income_required": 1500,
    "max_debt_to_income_ratio": 0.35,
    "review_threshold": 80000,
    "min_credit_score": 600
}'::jsonb),
('PT', 'Portugal', 'EUR', 'Europe/Lisbon', '{
    "min_loan_amount": 1000,
    "max_loan_amount": 75000,
    "min_income_required": 700,
    "max_debt_to_income_ratio": 0.35,
    "review_threshold": 25000,
    "min_credit_score": 650
}'::jsonb),
('IT', 'Italia', 'EUR', 'Europe/Rome', '{
    "min_loan_amount": 1000,
    "max_loan_amount": 80000,
    "min_income_required": 800,
    "max_debt_to_income_ratio": 0.35,
    "review_threshold": 30000,
    "min_credit_score": 650
}'::jsonb);

-- =====================================================
-- DATOS INICIALES: Tipos de documentos
-- =====================================================
INSERT INTO document_types (country_id, code, name, validation_regex) VALUES
((SELECT id FROM countries WHERE code = 'ES'), 'DNI', 'Documento Nacional de Identidad', '^[0-9]{8}[A-Z]$'),
((SELECT id FROM countries WHERE code = 'ES'), 'NIE', 'Número de Identidad de Extranjero', '^[XYZ][0-9]{7}[A-Z]$'),
((SELECT id FROM countries WHERE code = 'MX'), 'CURP', 'Clave Única de Registro de Población', '^[A-Z]{4}[0-9]{6}[HM][A-Z]{5}[0-9A-Z][0-9]$'),
((SELECT id FROM countries WHERE code = 'MX'), 'RFC', 'Registro Federal de Contribuyentes', '^[A-Z&Ñ]{3,4}[0-9]{6}[A-Z0-9]{3}$'),
((SELECT id FROM countries WHERE code = 'CO'), 'CC', 'Cédula de Ciudadanía', '^[0-9]{6,10}$'),
((SELECT id FROM countries WHERE code = 'CO'), 'CE', 'Cédula de Extranjería', '^[0-9]{6,10}$'),
((SELECT id FROM countries WHERE code = 'BR'), 'CPF', 'Cadastro de Pessoas Físicas', '^[0-9]{11}$'),
((SELECT id FROM countries WHERE code = 'BR'), 'RG', 'Registro Geral', '^[0-9]{7,9}$'),
((SELECT id FROM countries WHERE code = 'PT'), 'NIF', 'Número de Identificação Fiscal', '^[0-9]{9}$'),
((SELECT id FROM countries WHERE code = 'PT'), 'CC', 'Cartão de Cidadão', '^[0-9]{8}$'),
((SELECT id FROM countries WHERE code = 'IT'), 'CF', 'Codice Fiscale', '^[A-Z]{6}[0-9]{2}[A-Z][0-9]{2}[A-Z][0-9]{3}[A-Z]$'),
((SELECT id FROM countries WHERE code = 'IT'), 'CI', 'Carta d''Identità', '^[A-Z]{2}[0-9]{5}[A-Z]{2}$');

-- =====================================================
-- DATOS INICIALES: Reglas por país
-- =====================================================

-- España
INSERT INTO country_rules (country_id, rule_type, name, description, priority, config) VALUES
((SELECT id FROM countries WHERE code = 'ES'), 'DOCUMENT_VALIDATION', 'Validación DNI España', 'Verifica formato y letra de control del DNI', 100, '{
    "required_document": "DNI",
    "validate_checksum": true
}'::jsonb),
((SELECT id FROM countries WHERE code = 'ES'), 'AMOUNT_THRESHOLD', 'Umbral de revisión España', 'Solicitudes superiores a 30000€ requieren revisión', 90, '{
    "threshold": 30000,
    "action": "REQUIRE_REVIEW"
}'::jsonb),
((SELECT id FROM countries WHERE code = 'ES'), 'INCOME_CHECK', 'Verificación de ingresos España', 'El monto solicitado no puede superar 6x el ingreso mensual', 80, '{
    "max_income_multiplier": 6
}'::jsonb);

-- México  
INSERT INTO country_rules (country_id, rule_type, name, description, priority, config) VALUES
((SELECT id FROM countries WHERE code = 'MX'), 'DOCUMENT_VALIDATION', 'Validación CURP México', 'Verifica formato del CURP', 100, '{
    "required_document": "CURP",
    "validate_checksum": true
}'::jsonb),
((SELECT id FROM countries WHERE code = 'MX'), 'DEBT_RATIO', 'Relación deuda-ingreso México', 'La relación deuda/ingreso no puede superar 40%', 90, '{
    "max_ratio": 0.40
}'::jsonb),
((SELECT id FROM countries WHERE code = 'MX'), 'INCOME_CHECK', 'Verificación de ingresos México', 'El monto solicitado no puede superar 8x el ingreso mensual', 80, '{
    "max_income_multiplier": 8
}'::jsonb);

-- Colombia
INSERT INTO country_rules (country_id, rule_type, name, description, priority, config) VALUES
((SELECT id FROM countries WHERE code = 'CO'), 'DOCUMENT_VALIDATION', 'Validación CC Colombia', 'Verifica formato de la Cédula de Ciudadanía', 100, '{
    "required_document": "CC",
    "min_length": 6,
    "max_length": 10
}'::jsonb),
((SELECT id FROM countries WHERE code = 'CO'), 'DEBT_RATIO', 'Relación deuda-ingreso Colombia', 'La deuda total no puede superar 45% del ingreso', 90, '{
    "max_ratio": 0.45,
    "include_existing_debt": true
}'::jsonb);

-- Brasil
INSERT INTO country_rules (country_id, rule_type, name, description, priority, config) VALUES
((SELECT id FROM countries WHERE code = 'BR'), 'DOCUMENT_VALIDATION', 'Validación CPF Brasil', 'Verifica formato y dígitos verificadores del CPF', 100, '{
    "required_document": "CPF",
    "validate_checksum": true
}'::jsonb),
((SELECT id FROM countries WHERE code = 'BR'), 'CREDIT_SCORE', 'Score crediticio Brasil', 'Score mínimo de 600 puntos', 90, '{
    "min_score": 600,
    "provider": "SERASA"
}'::jsonb);

-- Portugal
INSERT INTO country_rules (country_id, rule_type, name, description, priority, config) VALUES
((SELECT id FROM countries WHERE code = 'PT'), 'DOCUMENT_VALIDATION', 'Validación NIF Portugal', 'Verifica formato del NIF', 100, '{
    "required_document": "NIF",
    "validate_checksum": true
}'::jsonb),
((SELECT id FROM countries WHERE code = 'PT'), 'INCOME_CHECK', 'Verificación de ingresos Portugal', 'El monto no puede superar 5x el ingreso mensual', 90, '{
    "max_income_multiplier": 5
}'::jsonb);

-- Italia
INSERT INTO country_rules (country_id, rule_type, name, description, priority, config) VALUES
((SELECT id FROM countries WHERE code = 'IT'), 'DOCUMENT_VALIDATION', 'Validación Codice Fiscale Italia', 'Verifica formato del Codice Fiscale', 100, '{
    "required_document": "CF",
    "validate_checksum": true
}'::jsonb),
((SELECT id FROM countries WHERE code = 'IT'), 'INCOME_CHECK', 'Estabilidad financiera Italia', 'Requiere mínimo 6 meses de empleo estable', 90, '{
    "min_months_employed": 6
}'::jsonb);

-- =====================================================
-- DATOS INICIALES: Proveedores bancarios
-- =====================================================
INSERT INTO banking_providers (country_id, code, name, type, config) VALUES
((SELECT id FROM countries WHERE code = 'ES'), 'ES_EQUIFAX', 'Equifax España', 'CREDIT_BUREAU', '{
    "base_url": "https://api.equifax.es",
    "timeout_seconds": 30,
    "retry_attempts": 3,
    "auth_type": "OAUTH2"
}'::jsonb),
((SELECT id FROM countries WHERE code = 'MX'), 'MX_BURO', 'Buró de Crédito México', 'CREDIT_BUREAU', '{
    "base_url": "https://api.burodecredito.com.mx",
    "timeout_seconds": 30,
    "retry_attempts": 3,
    "auth_type": "API_KEY"
}'::jsonb),
((SELECT id FROM countries WHERE code = 'CO'), 'CO_DATACREDITO', 'DataCrédito Colombia', 'CREDIT_BUREAU', '{
    "base_url": "https://api.datacredito.com.co",
    "timeout_seconds": 30,
    "retry_attempts": 3,
    "auth_type": "API_KEY"
}'::jsonb),
((SELECT id FROM countries WHERE code = 'BR'), 'BR_SERASA', 'Serasa Experian Brasil', 'CREDIT_BUREAU', '{
    "base_url": "https://api.serasa.com.br",
    "timeout_seconds": 30,
    "retry_attempts": 3,
    "auth_type": "OAUTH2"
}'::jsonb),
((SELECT id FROM countries WHERE code = 'PT'), 'PT_BDP', 'Banco de Portugal', 'CREDIT_BUREAU', '{
    "base_url": "https://api.bportugal.pt",
    "timeout_seconds": 30,
    "retry_attempts": 3,
    "auth_type": "OAUTH2"
}'::jsonb),
((SELECT id FROM countries WHERE code = 'IT'), 'IT_CRIF', 'CRIF Italia', 'CREDIT_BUREAU', '{
    "base_url": "https://api.crif.it",
    "timeout_seconds": 30,
    "retry_attempts": 3,
    "auth_type": "API_KEY"
}'::jsonb);

-- =====================================================
-- DATOS INICIALES: Usuario administrador
-- =====================================================
-- Password: admin123 (bcrypt hash)
INSERT INTO users (email, password_hash, full_name, role) VALUES
('admin@fintech.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'Administrador Sistema', 'ADMIN');

-- =====================================================
-- COMENTARIOS PARA DOCUMENTACIÓN
-- =====================================================
COMMENT ON TABLE countries IS 'Países donde opera la fintech. Sistema diseñado para N países configurables.';
COMMENT ON TABLE credit_applications IS 'Solicitudes de crédito. Tabla principal diseñada para millones de registros.';
COMMENT ON TABLE jobs_queue IS 'Cola de trabajos basada en PostgreSQL para procesamiento asíncrono.';
COMMENT ON TABLE audit_logs IS 'Registro de auditoría para todas las operaciones del sistema.';
COMMENT ON COLUMN credit_applications.validation_results IS 'Resultados de validación en formato JSON para flexibilidad.';
COMMENT ON COLUMN country_rules.config IS 'Configuración flexible de reglas en JSON para extensibilidad.';

