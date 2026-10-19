ALTER TABLE notification_logs ADD COLUMN IF NOT EXISTS severity VARCHAR(10) NOT NULL DEFAULT 'INFO'
    CHECK (severity IN ('INFO','WARNING','ERROR'));
ALTER TABLE notification_logs ADD COLUMN IF NOT EXISTS account_id UUID,
    ADD COLUMN IF NOT EXISTS details JSONB NOT NULL DEFAULT '{}';
CREATE INDEX IF NOT EXISTS idx_notifications_type_time ON notification_logs(notification_type,sent_at DESC);
