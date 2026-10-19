DROP TABLE customer_lifecycle_events;
ALTER TABLE customers
    DROP COLUMN lifecycle_correlation_id,
    DROP COLUMN lifecycle_trace_context,
    DROP COLUMN lifecycle_observed_from_creation,
    DROP COLUMN kyc_updated_at,
    DROP COLUMN kyc_verified_at;
