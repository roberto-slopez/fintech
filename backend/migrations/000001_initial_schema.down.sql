-- =====================================================
-- Fintech Multipaís - Rollback del esquema inicial
-- =====================================================

-- Eliminar triggers
DROP TRIGGER IF EXISTS trigger_notify_application_change ON credit_applications;
DROP TRIGGER IF EXISTS trigger_application_status_changed ON credit_applications;
DROP TRIGGER IF EXISTS trigger_application_created ON credit_applications;
DROP TRIGGER IF EXISTS update_jobs_queue_updated_at ON jobs_queue;
DROP TRIGGER IF EXISTS update_credit_applications_updated_at ON credit_applications;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TRIGGER IF EXISTS update_banking_providers_updated_at ON banking_providers;
DROP TRIGGER IF EXISTS update_country_rules_updated_at ON country_rules;
DROP TRIGGER IF EXISTS update_countries_updated_at ON countries;

-- Eliminar funciones
DROP FUNCTION IF EXISTS notify_application_change();
DROP FUNCTION IF EXISTS on_application_status_changed();
DROP FUNCTION IF EXISTS on_application_created();
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Eliminar tablas en orden correcto (por foreign keys)
DROP TABLE IF EXISTS webhook_events;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS jobs_queue;
DROP TABLE IF EXISTS state_transitions;
DROP TABLE IF EXISTS banking_info;
DROP TABLE IF EXISTS credit_applications;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS banking_providers;
DROP TABLE IF EXISTS country_rules;
DROP TABLE IF EXISTS document_types;
DROP TABLE IF EXISTS countries;

-- Eliminar extensiones
DROP EXTENSION IF EXISTS pg_trgm;
DROP EXTENSION IF EXISTS "uuid-ossp";

