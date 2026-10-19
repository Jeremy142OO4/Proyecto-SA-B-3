DROP INDEX IF EXISTS idx_transferencias_trace_id;
ALTER TABLE transferencias DROP COLUMN IF EXISTS trace_id, DROP COLUMN IF EXISTS trace_context;
