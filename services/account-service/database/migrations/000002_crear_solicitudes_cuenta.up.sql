CREATE TABLE solicitudes_creacion_cuenta (
    id_solicitud UUID PRIMARY KEY,
    id_cliente UUID NOT NULL,
    tipo_cuenta VARCHAR(20) NOT NULL,
    estado VARCHAR(20) NOT NULL DEFAULT 'PENDIENTE_VALIDACION',
    id_correlacion UUID NOT NULL,
    id_cuenta UUID,
    motivo_rechazo VARCHAR(255),
    fecha_creacion TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    fecha_actualizacion TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_solicitudes_correlacion UNIQUE (id_correlacion),
    CONSTRAINT fk_solicitudes_cuenta FOREIGN KEY (id_cuenta) REFERENCES cuentas (id_cuenta),
    CONSTRAINT ck_solicitudes_tipo CHECK (tipo_cuenta IN ('MONETARIA', 'AHORRO')),
    CONSTRAINT ck_solicitudes_estado CHECK (
        estado IN ('PENDIENTE_VALIDACION', 'COMPLETADA', 'RECHAZADA')
    )
);

CREATE INDEX idx_solicitudes_cliente ON solicitudes_creacion_cuenta (id_cliente);
CREATE INDEX idx_solicitudes_estado ON solicitudes_creacion_cuenta (estado, fecha_creacion);
