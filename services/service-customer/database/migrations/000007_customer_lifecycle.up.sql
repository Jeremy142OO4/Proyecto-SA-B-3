ALTER TABLE customers
    ADD COLUMN IF NOT EXISTS lifecycle_correlation_id UUID,
    ADD COLUMN IF NOT EXISTS lifecycle_trace_context TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS lifecycle_observed_from_creation BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS kyc_updated_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS kyc_verified_at TIMESTAMPTZ;

-- No inventar fechas de verificacion ni trazas para los clientes anteriores.
CREATE TABLE IF NOT EXISTS customer_lifecycle_events (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customers(customer_id),
    event_type VARCHAR(100) NOT NULL,
    state_kind VARCHAR(20) NOT NULL,
    previous_state VARCHAR(30) NOT NULL DEFAULT '',
    state VARCHAR(30) NOT NULL,
    correlation_id UUID NOT NULL,
    trace_context TEXT NOT NULL DEFAULT '',
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX IF NOT EXISTS idx_customer_lifecycle_customer_time
    ON customer_lifecycle_events(customer_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_customer_lifecycle_correlation
    ON customer_lifecycle_events(correlation_id);
