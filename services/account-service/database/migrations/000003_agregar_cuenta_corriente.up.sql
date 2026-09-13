ALTER TABLE cuentas DROP CONSTRAINT IF EXISTS ck_cuentas_tipo;
ALTER TABLE cuentas
    ADD CONSTRAINT ck_cuentas_tipo
    CHECK (tipo_cuenta IN ('MONETARIA', 'AHORRO', 'CORRIENTE'));

ALTER TABLE solicitudes_creacion_cuenta DROP CONSTRAINT IF EXISTS ck_solicitudes_tipo;
ALTER TABLE solicitudes_creacion_cuenta
    ADD CONSTRAINT ck_solicitudes_tipo
    CHECK (tipo_cuenta IN ('MONETARIA', 'AHORRO', 'CORRIENTE'));
