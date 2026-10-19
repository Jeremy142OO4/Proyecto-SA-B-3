CREATE TABLE IF NOT EXISTS cancelaciones_interbancarias (
 id UUID PRIMARY KEY, banco TEXT NOT NULL, creado TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE IF NOT EXISTS reversos_interbancarios (
 id UUID PRIMARY KEY, original UUID NOT NULL UNIQUE REFERENCES depositos_interbancarios(id),
 direccion TEXT NOT NULL CHECK(direccion IN ('IN','OUT')),
 solicitud JSONB NOT NULL, estado TEXT NOT NULL DEFAULT 'PROCESSING' CHECK(estado IN ('PROCESSING','COMPLETED','FAILED')),
 refund_id UUID UNIQUE, detalle TEXT NOT NULL DEFAULT '',
 proximo TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(), intentos INTEGER NOT NULL DEFAULT 0,
 trace_context TEXT NOT NULL DEFAULT '', actualizado TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
ALTER TABLE depositos_interbancarios ADD COLUMN IF NOT EXISTS refund_of UUID REFERENCES depositos_interbancarios(id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_deposito_devolucion ON depositos_interbancarios(refund_of) WHERE refund_of IS NOT NULL;
CREATE TABLE IF NOT EXISTS incidentes_interbancarios (
 transferencia UUID NOT NULL, tipo TEXT NOT NULL, fecha TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(transferencia,tipo)
);
CREATE TABLE IF NOT EXISTS conciliaciones_interbancarias (
 fecha DATE PRIMARY KEY, estado TEXT NOT NULL DEFAULT 'PENDING', pagina INTEGER NOT NULL DEFAULT 1,
 proximo TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(), discrepancias INTEGER NOT NULL DEFAULT 0,
 actualizado TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE IF NOT EXISTS conciliacion_items_interbancarios (
 fecha DATE NOT NULL REFERENCES conciliaciones_interbancarias(fecha), id UUID NOT NULL,
 contenido JSONB NOT NULL, PRIMARY KEY(fecha,id)
);
