-- Rollback authentication fields from company_users table

-- Drop indexes
DROP INDEX IF EXISTS idx_company_users_email;
DROP INDEX IF EXISTS idx_company_users_google_id;
DROP INDEX IF EXISTS idx_company_users_refresh_token;

-- Drop columns
ALTER TABLE company_users DROP COLUMN IF EXISTS email;
ALTER TABLE company_users DROP COLUMN IF EXISTS password_hash;
ALTER TABLE company_users DROP COLUMN IF EXISTS google_id;
ALTER TABLE company_users DROP COLUMN IF EXISTS oauth_provider;
ALTER TABLE company_users DROP COLUMN IF EXISTS avatar_url;
ALTER TABLE company_users DROP COLUMN IF EXISTS is_active;
ALTER TABLE company_users DROP COLUMN IF EXISTS email_verified;
ALTER TABLE company_users DROP COLUMN IF EXISTS last_login_at;
ALTER TABLE company_users DROP COLUMN IF EXISTS refresh_token;
ALTER TABLE company_users DROP COLUMN IF EXISTS refresh_token_expires_at;
