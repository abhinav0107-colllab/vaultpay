CREATE TABLE IF NOT EXISTS payment_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID NOT NULL,
    event_type VARCHAR(50) NOT NULL, -- e.g., 'payment_started', 'payment_success', 'payment_failed'
    previous_state VARCHAR(50),
    current_state VARCHAR(50) NOT NULL,
    payload JSONB,                    -- Stores custom snapshot metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP    
);

-- Index the payment_id so we can look up histories quickly
CREATE INDEX IF NOT EXISTS idx_payment_events_payment_id ON payment_events(payment_id);