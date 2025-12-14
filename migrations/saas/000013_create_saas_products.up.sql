-- Create products table for SaaS module
CREATE TABLE IF NOT EXISTS saas_products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,

    -- Product Info
    name TEXT NOT NULL,
    description TEXT,
    sku TEXT, -- Stock Keeping Unit / Product Code
    category TEXT,

    -- Pricing & Stock
    price DECIMAL(12,2) NOT NULL DEFAULT 0,
    stock INTEGER NOT NULL DEFAULT 0,

    -- Media
    image_url TEXT, -- For now just URL, Phase 3 will add upload

    -- Status
    is_active BOOLEAN DEFAULT true,

    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP -- Soft delete support
);

-- Create indexes for better performance
CREATE INDEX idx_saas_products_client_id ON saas_products(client_id);
CREATE INDEX idx_saas_products_sku ON saas_products(sku);
CREATE INDEX idx_saas_products_category ON saas_products(category);
CREATE INDEX idx_saas_products_is_active ON saas_products(is_active);
CREATE INDEX idx_saas_products_deleted_at ON saas_products(deleted_at);

-- Create unique constraint for SKU per client (optional, but recommended)
CREATE UNIQUE INDEX idx_saas_products_client_sku ON saas_products(client_id, sku) WHERE sku IS NOT NULL AND deleted_at IS NULL;

-- Add comment to table
COMMENT ON TABLE saas_products IS 'Products catalog for SaaS module - multi-tenant support';
