-- Seed data for testing SAAS module
-- Run this manually after migrations: psql -d your_database -f migrations/saas/seed_data.sql

-- Insert sample client
INSERT INTO saas_clients (whatsapp_number, business_name, subscription_plan, subscription_status, tone)
VALUES ('628123456789', 'Agung Store', 'pro', 'active', 'friendly')
ON CONFLICT DO NOTHING;

-- Get the client ID (you may need to adjust this based on your UUID)
-- For demo purposes, we'll use a specific UUID
DO $$
DECLARE
    client_uuid UUID;
BEGIN
    -- Get or create client
    SELECT id INTO client_uuid FROM saas_clients WHERE whatsapp_number = '628123456789' LIMIT 1;

    -- Insert FAQ knowledge base entries
    INSERT INTO saas_knowledge_base (client_id, type, title, content, tags)
    VALUES
    (
        client_uuid,
        'faq',
        'Cara Order',
        '{"question": "Bagaimana cara order?", "answer": "Silakan ketik ORDER [nama produk] [jumlah]. Contoh: ORDER Kopi Arabica 2"}'::jsonb,
        ARRAY['order', 'howto']
    ),
    (
        client_uuid,
        'faq',
        'Jam Operasional',
        '{"question": "Jam berapa toko buka?", "answer": "Kami buka setiap hari Senin-Jumat pukul 09:00-17:00 WIB"}'::jsonb,
        ARRAY['hours', 'info']
    )
    ON CONFLICT DO NOTHING;

    -- Insert Product knowledge base entries
    INSERT INTO saas_knowledge_base (client_id, type, title, content, tags)
    VALUES
    (
        client_uuid,
        'product',
        'Kopi Arabica Premium',
        '{"name": "Kopi Arabica Premium", "price": 50000, "description": "Kopi pilihan dari petani lokal", "stock": 100}'::jsonb,
        ARRAY['coffee', 'premium']
    ),
    (
        client_uuid,
        'product',
        'Kopi Robusta',
        '{"name": "Kopi Robusta", "price": 35000, "description": "Kopi robusta berkualitas tinggi", "stock": 150}'::jsonb,
        ARRAY['coffee']
    )
    ON CONFLICT DO NOTHING;

    -- Insert credit record
    INSERT INTO saas_credits (client_id, credits_used, period_start, period_end)
    VALUES (
        client_uuid,
        0,
        CURRENT_DATE,
        CURRENT_DATE + INTERVAL '30 days'
    )
    ON CONFLICT DO NOTHING;

END $$;

-- Verify data
SELECT 'Clients:' as table_name, COUNT(*) as count FROM saas_clients
UNION ALL
SELECT 'Knowledge Base:', COUNT(*) FROM saas_knowledge_base
UNION ALL
SELECT 'Credits:', COUNT(*) FROM saas_credits;
