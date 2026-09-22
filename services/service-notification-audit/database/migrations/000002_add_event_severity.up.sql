ALTER TABLE audit_logs
    ADD COLUMN IF NOT EXISTS severity VARCHAR(20) NOT NULL DEFAULT 'INFO'
    CHECK (severity IN ('INFO', 'WARNING', 'ERROR'));

CREATE INDEX IF NOT EXISTS idx_audit_severity ON audit_logs(severity);
