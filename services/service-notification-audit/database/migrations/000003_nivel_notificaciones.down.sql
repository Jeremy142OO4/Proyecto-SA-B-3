ALTER TABLE notification_logs DROP COLUMN severity;
DROP INDEX idx_notifications_type_time;
ALTER TABLE notification_logs DROP COLUMN account_id, DROP COLUMN details;
