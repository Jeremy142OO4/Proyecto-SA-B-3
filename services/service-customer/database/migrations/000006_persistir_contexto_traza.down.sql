-- Roll back application code before dropping telemetry metadata.
ALTER TABLE outbox_messages DROP COLUMN trace_context;
