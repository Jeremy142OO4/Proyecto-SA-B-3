-- Roll back application code before dropping telemetry metadata.
ALTER TABLE mensajes_salida DROP COLUMN trace_context;
