-- 1. Create the Users table
-- We use BIGINT for balances to store "minor units" (paise/cents) to avoid float rounding errors.
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE NOT NULL,
    balance BIGINT NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'INR',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 2. Create the API Keys table
-- Links to a user. We store a HASH of the key for security.
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_hash TEXT UNIQUE NOT NULL,
    label TEXT, -- e.g., "Production Key" or "Test Key"
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 3. Create the Payments table
-- This is the heart of the system.
CREATE TYPE payment_status AS ENUM ('pending', 'processing', 'succeeded', 'failed', 'refunded');

CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    amount BIGINT NOT NULL,
    currency TEXT NOT NULL,
    status payment_status NOT NULL DEFAULT 'pending',
    idempotency_key TEXT UNIQUE, -- Prevents double-charging
    provider_tx_id TEXT, -- The ID from Razorpay/Stripe
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexing for speed
CREATE INDEX idx_payments_user_id ON payments(user_id);
CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE TABLE IF NOT EXISTS payment_events (
    id SERIAL PRIMARY KEY,
    payment_id VARCHAR(50) NOT NULL,
    event_type VARCHAR(50) NOT NULL, -- e.g., "PAYMENT_CREATED", "PAYMENT_AUTHORIZED", "PAYMENT_FAILED"
    payload JSONB NOT NULL,          -- Stores full metadata snapshot at that exact second
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_payment_events_payment_id ON payment_events(payment_id);