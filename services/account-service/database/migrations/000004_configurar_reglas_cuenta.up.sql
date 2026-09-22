ALTER TABLE cuentas
    ADD COLUMN saldo_minimo_centavos BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN comision_transaccion_centavos BIGINT NOT NULL DEFAULT 0;

ALTER TABLE cuentas
    ADD CONSTRAINT ck_cuentas_saldo_minimo_no_negativo CHECK (saldo_minimo_centavos >= 0),
    ADD CONSTRAINT ck_cuentas_comision_no_negativa CHECK (comision_transaccion_centavos >= 0);

ALTER TABLE solicitudes_creacion_cuenta
    ADD COLUMN saldo_minimo_centavos BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN comision_transaccion_centavos BIGINT NOT NULL DEFAULT 0;

ALTER TABLE solicitudes_creacion_cuenta
    ADD CONSTRAINT ck_solicitudes_saldo_minimo_no_negativo CHECK (saldo_minimo_centavos >= 0),
    ADD CONSTRAINT ck_solicitudes_comision_no_negativa CHECK (comision_transaccion_centavos >= 0);
