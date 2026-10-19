CREATE TABLE IF NOT EXISTS depositos_interbancarios (
 id UUID PRIMARY KEY,
 direccion TEXT NOT NULL CHECK (direccion IN ('IN','OUT')),
 cliente TEXT NOT NULL DEFAULT '',
 solicitud JSONB NOT NULL,
 huella TEXT NOT NULL,
 estado TEXT NOT NULL CHECK(estado IN ('PENDING_RESERVATION','RESERVED','PROCESSING','TIMEOUT','ACCEPTED','REJECTED','REVERSED','CANCELLED')),
 motivo TEXT NOT NULL DEFAULT '',
 resultado JSONB,
 trace_context TEXT NOT NULL DEFAULT '',
 recibido TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 actualizado TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 proximo TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 intentos INTEGER NOT NULL DEFAULT 0,
 callback_entregado BOOLEAN NOT NULL DEFAULT false
);
CREATE INDEX IF NOT EXISTS idx_depositos_interbancarios_trabajo ON depositos_interbancarios(proximo);
