-- Additive: existing rows and previous application versions remain valid.
ALTER TABLE outbox_messages ADD COLUMN IF NOT EXISTS trace_context TEXT NOT NULL DEFAULT '';
