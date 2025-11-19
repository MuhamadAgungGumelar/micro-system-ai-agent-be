-- Remove session_id column from saas_clients table

DROP INDEX IF EXISTS idx_saas_clients_session_id;

ALTER TABLE saas_clients
DROP COLUMN IF EXISTS session_id;
