-- Additive: existing rows and previous application versions remain valid.
ALTER TABLE mensajes_salida ADD COLUMN IF NOT EXISTS trace_context TEXT NOT NULL DEFAULT '';
