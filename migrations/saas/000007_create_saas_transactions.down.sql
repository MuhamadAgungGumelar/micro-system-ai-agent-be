-- Drop trigger first
DROP TRIGGER IF EXISTS trigger_update_saas_transactions_updated_at ON saas_transactions;
DROP FUNCTION IF EXISTS update_saas_transactions_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_saas_transactions_client;
DROP INDEX IF EXISTS idx_saas_transactions_date;
DROP INDEX IF EXISTS idx_saas_transactions_created_from;
DROP INDEX IF EXISTS idx_saas_transactions_source_type;

-- Drop table
DROP TABLE IF EXISTS saas_transactions;
