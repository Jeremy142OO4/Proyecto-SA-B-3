CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE cuentas (
    id_cuenta UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_cliente UUID NOT NULL,
    numero_cuenta VARCHAR(20) NOT NULL,
    tipo_cuenta VARCHAR(20) NOT NULL,
    saldo_centavos BIGINT NOT NULL DEFAULT 0,
    moneda CHAR(3) NOT NULL DEFAULT 'GTQ',
    estado VARCHAR(20) NOT NULL DEFAULT 'ACTIVA',
    ultima_actividad TIMESTAMPTZ,
    fecha_creacion TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    fecha_actualizacion TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    version BIGINT NOT NULL DEFAULT 1,
    CONSTRAINT uq_cuentas_numero_cuenta UNIQUE (numero_cuenta),
    CONSTRAINT ck_cuentas_tipo CHECK (tipo_cuenta IN ('MONETARIA', 'AHORRO')),
    CONSTRAINT ck_cuentas_estado CHECK (estado IN ('ACTIVA', 'INACTIVA', 'BLOQUEADA', 'CERRADA')),
    CONSTRAINT ck_cuentas_saldo_no_negativo CHECK (saldo_centavos >= 0),
    CONSTRAINT ck_cuentas_moneda CHECK (moneda = 'GTQ')
);

CREATE INDEX idx_cuentas_id_cliente ON cuentas (id_cliente);
CREATE INDEX idx_cuentas_estado_ultima_actividad ON cuentas (estado, ultima_actividad);

CREATE TABLE movimientos_cuenta (
    id_movimiento UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_cuenta UUID NOT NULL,
    id_operacion UUID NOT NULL,
    id_correlacion UUID NOT NULL,
    tipo_movimiento VARCHAR(20) NOT NULL,
    monto_centavos BIGINT NOT NULL,
    saldo_anterior_centavos BIGINT NOT NULL,
    saldo_nuevo_centavos BIGINT NOT NULL,
    descripcion VARCHAR(255),
    fecha_creacion TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_movimientos_cuenta
        FOREIGN KEY (id_cuenta) REFERENCES cuentas (id_cuenta),
    CONSTRAINT uq_movimientos_operacion_tipo UNIQUE (id_operacion, tipo_movimiento),
    CONSTRAINT ck_movimientos_tipo CHECK (tipo_movimiento IN ('DEBITO', 'CREDITO', 'COMPENSACION')),
    CONSTRAINT ck_movimientos_monto_positivo CHECK (monto_centavos > 0),
    CONSTRAINT ck_movimientos_saldos_no_negativos
        CHECK (saldo_anterior_centavos >= 0 AND saldo_nuevo_centavos >= 0)
);

CREATE INDEX idx_movimientos_id_cuenta_fecha
    ON movimientos_cuenta (id_cuenta, fecha_creacion DESC);
CREATE INDEX idx_movimientos_id_correlacion
    ON movimientos_cuenta (id_correlacion);

CREATE TABLE mensajes_procesados (
    id_mensaje UUID NOT NULL,
    nombre_consumidor VARCHAR(100) NOT NULL,
    tipo_mensaje VARCHAR(100) NOT NULL,
    id_correlacion UUID NOT NULL,
    fecha_procesamiento TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resultado JSONB,
    PRIMARY KEY (id_mensaje, nombre_consumidor)
);

CREATE INDEX idx_mensajes_procesados_correlacion
    ON mensajes_procesados (id_correlacion);

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
    CONSTRAINT ck_mensajes_salida_version CHECK (version_evento > 0),
    CONSTRAINT ck_mensajes_salida_intentos CHECK (cantidad_intentos >= 0),
    CONSTRAINT ck_mensajes_salida_estado
        CHECK (estado IN ('PENDIENTE', 'PUBLICADO', 'FALLIDO'))
);

CREATE INDEX idx_mensajes_salida_pendientes
    ON mensajes_salida (fecha_creacion)
    WHERE estado = 'PENDIENTE';
