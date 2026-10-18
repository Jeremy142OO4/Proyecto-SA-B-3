-- Apply before deploying the instrumented publisher.
ALTER TABLE mensajes_salida ADD COLUMN IF NOT EXISTS trace_context TEXT NOT NULL DEFAULT '';
