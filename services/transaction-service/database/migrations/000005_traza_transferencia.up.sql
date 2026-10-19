-- Additive and repeatable for the CI/CD schema gate.
ALTER TABLE transferencias ADD COLUMN IF NOT EXISTS trace_id VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS trace_context TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_transferencias_trace_id ON transferencias(trace_id) WHERE trace_id <> '';
