DROP INDEX IF EXISTS idx_audit_severity;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS severity;
