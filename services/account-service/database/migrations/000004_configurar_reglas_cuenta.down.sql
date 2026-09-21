ALTER TABLE solicitudes_creacion_cuenta
    DROP CONSTRAINT IF EXISTS ck_solicitudes_saldo_minimo_no_negativo,
    DROP CONSTRAINT IF EXISTS ck_solicitudes_comision_no_negativa,
    DROP COLUMN IF EXISTS saldo_minimo_centavos,
    DROP COLUMN IF EXISTS comision_transaccion_centavos;

ALTER TABLE cuentas
    DROP CONSTRAINT IF EXISTS ck_cuentas_saldo_minimo_no_negativo,
    DROP CONSTRAINT IF EXISTS ck_cuentas_comision_no_negativa,
    DROP COLUMN IF EXISTS saldo_minimo_centavos,
    DROP COLUMN IF EXISTS comision_transaccion_centavos;
