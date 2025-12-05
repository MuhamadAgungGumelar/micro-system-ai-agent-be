-- Add deleted_at column for soft deletes
ALTER TABLE saas_carts ADD COLUMN deleted_at TIMESTAMP;

-- Create index on deleted_at
CREATE INDEX idx_carts_deleted_at ON saas_carts(deleted_at);
