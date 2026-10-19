DROP INDEX IF EXISTS idx_audit_slo_service_time;
DROP INDEX IF EXISTS idx_audit_trace_time;
DROP INDEX IF EXISTS idx_audit_correlation_time;
DROP INDEX IF EXISTS idx_audit_service_level_time;
DROP INDEX IF EXISTS idx_audit_time_id;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS trace_id;
