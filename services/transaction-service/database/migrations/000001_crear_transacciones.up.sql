CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE TABLE transferencias(
 id_transferencia UUID PRIMARY KEY,
 id_cliente UUID NOT NULL,
 id_cuenta_origen UUID NOT NULL,
 id_cuenta_destino UUID NOT NULL,
 id_correlacion UUID NOT NULL,
 monto_centavos BIGINT NOT NULL CHECK(monto_centavos>0),
 moneda CHAR(3) NOT NULL DEFAULT 'GTQ' CHECK(moneda='GTQ'),
 descripcion VARCHAR(255) NOT NULL DEFAULT '',
 estado VARCHAR(30) NOT NULL CHECK(estado IN('PENDIENTE','PROCESANDO','COMPLETADA','RECHAZADA','COMPENSANDO','COMPENSADA','COMPENSACION_FALLIDA')),
 codigo_error VARCHAR(80) NOT NULL DEFAULT '',
 fecha_creacion TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 fecha_actualizacion TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 CHECK(id_cuenta_origen<>id_cuenta_destino)
);
CREATE INDEX idx_transferencias_cliente_fecha ON transferencias(id_cliente,fecha_creacion DESC);
CREATE UNIQUE INDEX idx_transferencias_correlacion ON transferencias(id_correlacion);
CREATE TABLE mensajes_procesados(id_mensaje UUID NOT NULL,nombre_consumidor VARCHAR(100) NOT NULL,tipo_mensaje VARCHAR(100) NOT NULL,id_correlacion UUID NOT NULL,fecha_procesamiento TIMESTAMPTZ NOT NULL DEFAULT NOW(),PRIMARY KEY(id_mensaje,nombre_consumidor));
CREATE TABLE mensajes_salida(id_mensaje UUID PRIMARY KEY,tipo_evento VARCHAR(100) NOT NULL,contenido JSONB NOT NULL,id_correlacion UUID NOT NULL,es_comando BOOLEAN NOT NULL,fecha_creacion TIMESTAMPTZ NOT NULL DEFAULT NOW(),fecha_publicacion TIMESTAMPTZ,cantidad_intentos INTEGER NOT NULL DEFAULT 0,estado VARCHAR(20) NOT NULL DEFAULT 'PENDIENTE' CHECK(estado IN('PENDIENTE','PUBLICADO','FALLIDO')));
CREATE INDEX idx_salida_pendiente ON mensajes_salida(fecha_creacion) WHERE estado='PENDIENTE';
