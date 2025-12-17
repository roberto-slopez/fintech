-- Migración 002 DOWN: Eliminar tablas adicionales

DROP TABLE IF EXISTS notifications CASCADE;
DROP TABLE IF EXISTS webhook_deliveries CASCADE;
DROP TABLE IF EXISTS webhook_endpoints CASCADE;
