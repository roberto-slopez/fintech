-- Rollback Migración 002

DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS state_transitions;
DROP TABLE IF EXISTS banking_info;
DROP TABLE IF EXISTS webhook_deliveries;
DROP TABLE IF EXISTS webhook_events;
DROP TABLE IF EXISTS webhook_endpoints;
DROP TABLE IF EXISTS audit_logs;

-- Remover columna risk_score si existe
ALTER TABLE credit_applications DROP COLUMN IF EXISTS risk_score;

