-- Add authentication fields to company_users table

-- Email for login
ALTER TABLE company_users ADD COLUMN IF NOT EXISTS email TEXT UNIQUE;

-- Password hash for email/password login
ALTER TABLE company_users ADD COLUMN IF NOT EXISTS password_hash TEXT;

-- OAuth fields
ALTER TABLE company_users ADD COLUMN IF NOT EXISTS google_id TEXT UNIQUE;
ALTER TABLE company_users ADD COLUMN IF NOT EXISTS oauth_provider TEXT DEFAULT 'email';

-- User profile
ALTER TABLE company_users ADD COLUMN IF NOT EXISTS avatar_url TEXT;

-- Status and tracking
ALTER TABLE company_users ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;
ALTER TABLE company_users ADD COLUMN IF NOT EXISTS email_verified BOOLEAN DEFAULT false;
ALTER TABLE company_users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMP;

-- Refresh token for JWT
ALTER TABLE company_users ADD COLUMN IF NOT EXISTS refresh_token TEXT;
ALTER TABLE company_users ADD COLUMN IF NOT EXISTS refresh_token_expires_at TIMESTAMP;

-- Add indexes for performance
CREATE INDEX IF NOT EXISTS idx_company_users_email ON company_users(email);
CREATE INDEX IF NOT EXISTS idx_company_users_google_id ON company_users(google_id);
CREATE INDEX IF NOT EXISTS idx_company_users_refresh_token ON company_users(refresh_token);

-- Add comment
COMMENT ON COLUMN company_users.email IS 'Email for authentication';
COMMENT ON COLUMN company_users.password_hash IS 'Bcrypt hashed password';
COMMENT ON COLUMN company_users.google_id IS 'Google OAuth user ID';
COMMENT ON COLUMN company_users.oauth_provider IS 'Authentication provider: email, google';
COMMENT ON COLUMN company_users.refresh_token IS 'JWT refresh token';
