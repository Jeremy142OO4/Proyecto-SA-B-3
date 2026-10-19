ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS trace_id VARCHAR(32) NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_audit_trace_time ON audit_logs(trace_id,occurred_at DESC,id DESC) WHERE trace_id<>'';
CREATE INDEX IF NOT EXISTS idx_audit_correlation_time ON audit_logs(correlation_id,occurred_at DESC,id DESC);
CREATE INDEX IF NOT EXISTS idx_audit_service_level_time ON audit_logs(producer,severity,occurred_at DESC,id DESC);
CREATE INDEX IF NOT EXISTS idx_audit_time_id ON audit_logs(occurred_at DESC,id DESC);
CREATE INDEX IF NOT EXISTS idx_audit_slo_service_time ON audit_logs((payload->>'servicio'),occurred_at DESC,id DESC) WHERE event_type='SLOAlertTriggered';
