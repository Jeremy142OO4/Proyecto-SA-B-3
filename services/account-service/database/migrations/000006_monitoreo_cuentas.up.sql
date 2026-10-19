CREATE TABLE IF NOT EXISTS configuracion_monitoreo_cuentas (
    id BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id),
    max_operaciones INTEGER NOT NULL CHECK (max_operaciones BETWEEN 1 AND 10000),
    ventana_minutos INTEGER NOT NULL CHECK (ventana_minutos BETWEEN 1 AND 1440),
    version BIGINT NOT NULL DEFAULT 1,
    id_administrador UUID NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE IF NOT EXISTS anomalias_cuenta (
    id_anomalia UUID PRIMARY KEY,
    id_cuenta UUID NOT NULL REFERENCES cuentas(id_cuenta),
    id_movimiento UUID NOT NULL UNIQUE REFERENCES movimientos_cuenta(id_movimiento),
    id_correlacion UUID NOT NULL,
    operaciones INTEGER NOT NULL,
    max_operaciones INTEGER NOT NULL,
    ventana_minutos INTEGER NOT NULL,
    version_configuracion BIGINT NOT NULL,
    detectada_en TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_anomalias_cuenta_fecha ON anomalias_cuenta(id_cuenta,detectada_en DESC);
