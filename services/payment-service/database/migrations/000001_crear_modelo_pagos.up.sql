CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE pagos (
    id_pago UUID PRIMARY KEY,
    id_cliente UUID NOT NULL,
    id_cuenta_origen UUID NOT NULL,
    beneficiario VARCHAR(150) NOT NULL,
    concepto VARCHAR(255) NOT NULL,
    monto_centavos BIGINT NOT NULL,
    moneda CHAR(3) NOT NULL DEFAULT 'GTQ',
    tipo_pago VARCHAR(20) NOT NULL,
    estado VARCHAR(20) NOT NULL DEFAULT 'PENDIENTE',
    referencia_externa VARCHAR(100),
    id_correlacion UUID NOT NULL,
    motivo_rechazo VARCHAR(255),
    fecha_creacion TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    fecha_actualizacion TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_pagos_monto CHECK (monto_centavos > 0),
    CONSTRAINT ck_pagos_moneda CHECK (moneda = 'GTQ'),
    CONSTRAINT ck_pagos_tipo CHECK (tipo_pago IN ('INTERNO','EXTERNO')),
    CONSTRAINT ck_pagos_estado CHECK (estado IN ('PENDIENTE','PROCESANDO','COMPENSANDO','COMPLETADO','RECHAZADO')),
    CONSTRAINT uq_pagos_correlacion UNIQUE (id_correlacion)
);

CREATE INDEX idx_pagos_cliente_fecha ON pagos (id_cliente,fecha_creacion DESC);
CREATE INDEX idx_pagos_cuenta_fecha ON pagos (id_cuenta_origen,fecha_creacion DESC);

CREATE TABLE intentos_pago (
    id_intento UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_pago UUID NOT NULL REFERENCES pagos(id_pago),
    numero_intento INTEGER NOT NULL,
    estado VARCHAR(20) NOT NULL,
    codigo_respuesta VARCHAR(50),
    detalle_error VARCHAR(255),
    fecha_inicio TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    fecha_finalizacion TIMESTAMPTZ,
    CONSTRAINT uq_intentos_pago_numero UNIQUE (id_pago,numero_intento),
    CONSTRAINT ck_intentos_numero CHECK (numero_intento > 0)
);

CREATE TABLE mensajes_procesados (
    id_mensaje UUID NOT NULL,
    nombre_consumidor VARCHAR(100) NOT NULL,
    tipo_mensaje VARCHAR(100) NOT NULL,
    id_correlacion UUID NOT NULL,
    fecha_procesamiento TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resultado JSONB,
    PRIMARY KEY (id_mensaje,nombre_consumidor)
);

CREATE TABLE mensajes_salida (
    id_mensaje UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tipo_evento VARCHAR(100) NOT NULL,
    version_evento INTEGER NOT NULL DEFAULT 1,
    contenido JSONB NOT NULL,
    id_correlacion UUID NOT NULL,
    fecha_creacion TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    fecha_publicacion TIMESTAMPTZ,
    cantidad_intentos INTEGER NOT NULL DEFAULT 0,
    estado VARCHAR(20) NOT NULL DEFAULT 'PENDIENTE',
    CONSTRAINT ck_salida_estado CHECK (estado IN ('PENDIENTE','PUBLICADO','FALLIDO'))
);
CREATE INDEX idx_salida_pagos_pendiente ON mensajes_salida(fecha_creacion) WHERE estado='PENDIENTE';
