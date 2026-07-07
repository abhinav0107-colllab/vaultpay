CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type VARCHAR(50) NOT NULL,    -- e.g., 'payment'
    aggregate_id VARCHAR(100) NOT NULL,     -- e.g., payment_id reference
    event_type VARCHAR(50) NOT NULL,        -- e.g., 'payment.succeeded'
    payload JSONB NOT NULL,                 -- Complete JSON metadata for the webhook
    status VARCHAR(20) DEFAULT 'PENDING',   -- PENDING, PROCESSED, FAILED
    retry_count INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP WITH TIME ZONE
);

-- Indexing strategies to optimize real-time polling engines
CREATE INDEX IF NOT EXISTS idx_outbox_pending ON outbox_events(status, created_at) WHERE status = 'PENDING'; 